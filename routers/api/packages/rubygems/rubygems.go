// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package rubygems

import (
	"compress/gzip"
	"compress/zlib"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	packages_model "forgejo.org/models/packages"
	"forgejo.org/modules/optional"
	packages_module "forgejo.org/modules/packages"
	rubygems_module "forgejo.org/modules/packages/rubygems"
	"forgejo.org/modules/util"
	"forgejo.org/routers/api/packages/helper"
	"forgejo.org/services/context"
	packages_service "forgejo.org/services/packages"
)

const (
	Sep = "---\n"
)

func apiError(ctx *context.Context, status int, obj any) {
	helper.LogAndProcessError(ctx, status, obj, func(message string) {
		ctx.PlainText(status, message)
	})
}

// EnumeratePackages serves the package list
func EnumeratePackages(ctx *context.Context) {
	packages, err := packages_model.GetVersionsByPackageType(ctx, ctx.Package.Owner.ID, packages_model.TypeRubyGems)
	if err != nil {
		apiError(ctx, http.StatusInternalServerError, err)
		return
	}

	enumeratePackages(ctx, "specs.4.8", packages)
}

// EnumeratePackagesLatest serves the list of the latest version of every package
func EnumeratePackagesLatest(ctx *context.Context) {
	pvs, _, err := packages_model.SearchLatestVersions(ctx, &packages_model.PackageSearchOptions{
		OwnerID:    ctx.Package.Owner.ID,
		Type:       packages_model.TypeRubyGems,
		IsInternal: optional.Some(false),
	})
	if err != nil {
		apiError(ctx, http.StatusInternalServerError, err)
		return
	}

	enumeratePackages(ctx, "latest_specs.4.8", pvs)
}

// EnumeratePackagesPreRelease is not supported and serves an empty list
func EnumeratePackagesPreRelease(ctx *context.Context) {
	enumeratePackages(ctx, "prerelease_specs.4.8", []*packages_model.PackageVersion{})
}

func enumeratePackages(ctx *context.Context, filename string, pvs []*packages_model.PackageVersion) {
	pds, err := packages_model.GetPackageDescriptors(ctx, pvs)
	if err != nil {
		apiError(ctx, http.StatusInternalServerError, err)
		return
	}

	specs := make([]any, 0, len(pds))
	for _, p := range pds {
		specs = append(specs, []any{
			p.Package.Name,
			&rubygems_module.RubyUserMarshal{
				Name:  "Gem::Version",
				Value: []string{p.Version.Version},
			},
			p.Metadata.(*rubygems_module.Metadata).Platform,
		})
	}

	ctx.SetServeHeaders(&context.ServeHeaderOptions{
		Filename: filename + ".gz",
	})

	zw := gzip.NewWriter(ctx.Resp)
	defer zw.Close()

	zw.Name = filename

	if err := rubygems_module.NewMarshalEncoder(zw).Encode(specs); err != nil {
		ctx.ServerError("Download file failed", err)
	}
}

// Serves info file for rubygems.org compatible /info/{gem} file.
// See also https://guides.rubygems.org/rubygems-org-compact-index-api/.
func ServePackageInfo(ctx *context.Context) {
	packageName := ctx.Params("package")
	versions, err := packages_model.GetVersionsByPackageName(
		ctx, ctx.Package.Owner.ID, packages_model.TypeRubyGems, packageName)
	if err != nil {
		apiError(ctx, http.StatusInternalServerError, err)
		return
	}
	if len(versions) == 0 {
		apiError(ctx, http.StatusNotFound, fmt.Sprintf("Could not find package %s", packageName))
		return
	}

	result, err := buildInfoFileForPackage(ctx, versions)
	if err != nil {
		apiError(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.PlainText(http.StatusOK, *result)
}

// ServeVersionsFile creates rubygems.org compatible /versions file.
// See also https://guides.rubygems.org/rubygems-org-compact-index-api/.
func ServeVersionsFile(ctx *context.Context) {
	packages, err := packages_model.GetPackagesByType(
		ctx, ctx.Package.Owner.ID, packages_model.TypeRubyGems)
	if err != nil {
		apiError(ctx, http.StatusInternalServerError, err)
		return
	}
	result := new(strings.Builder)
	result.WriteString(Sep)
	for _, pack := range packages {
		versions, err := packages_model.GetVersionsByPackageName(
			ctx, ctx.Package.Owner.ID, packages_model.TypeRubyGems, pack.Name)
		if err != nil {
			apiError(ctx, http.StatusInternalServerError, err)
			return
		}
		if len(versions) == 0 {
			// No versions left for this package, we should continue.
			continue
		}

		fmt.Fprintf(result, "%s ", pack.Name)
		for i, v := range versions {
			result.WriteString(v.Version)

			pd, err := packages_model.GetPackageDescriptor(ctx, v)
			if err != nil {
				apiError(ctx, http.StatusInternalServerError, err)
				return
			}

			metadata := pd.Metadata.(*rubygems_module.Metadata)

			if metadata.Platform != "ruby" {
				result.WriteString("_")
				result.WriteString(metadata.Platform)
			}

			if i != len(versions)-1 {
				result.WriteString(",")
			}
		}

		info, err := buildInfoFileForPackage(ctx, versions)
		if err != nil {
			apiError(ctx, http.StatusInternalServerError, err)
			return
		}

		checksum := md5.Sum([]byte(*info))
		fmt.Fprintf(result, " %x\n", checksum)
	}
	ctx.PlainText(http.StatusOK, result.String())
}

// ServePackageSpecification serves the compressed Gemspec file of a package
func ServePackageSpecification(ctx *context.Context) {
	filename := ctx.Params("filename")

	if !strings.HasSuffix(filename, ".gemspec.rz") {
		apiError(ctx, http.StatusNotImplemented, nil)
		return
	}

	pvs, err := getVersionsByFilename(ctx, filename[:len(filename)-10]+"gem")
	if err != nil {
		apiError(ctx, http.StatusInternalServerError, err)
		return
	}

	if len(pvs) != 1 {
		apiError(ctx, http.StatusNotFound, nil)
		return
	}

	pd, err := packages_model.GetPackageDescriptor(ctx, pvs[0])
	if err != nil {
		apiError(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.SetServeHeaders(&context.ServeHeaderOptions{
		Filename: filename,
	})

	zw := zlib.NewWriter(ctx.Resp)
	defer zw.Close()

	metadata := pd.Metadata.(*rubygems_module.Metadata)

	// create a Ruby Gem::Specification object
	spec := &rubygems_module.RubyUserDef{
		Name: "Gem::Specification",
		Value: []any{
			"3.2.3", // @rubygems_version
			4,       // @specification_version,
			pd.Package.Name,
			&rubygems_module.RubyUserMarshal{
				Name:  "Gem::Version",
				Value: []string{pd.Version.Version},
			},
			nil,               // date
			metadata.Summary,  // @summary
			nil,               // @required_ruby_version
			nil,               // @required_rubygems_version
			metadata.Platform, // @original_platform
			[]any{},           // @dependencies
			nil,               // rubyforge_project
			"",                // @email
			metadata.Authors,
			metadata.Description,
			metadata.ProjectURL,
			true,              // has_rdoc
			metadata.Platform, // @new_platform
			nil,
			metadata.Licenses,
		},
	}

	if err := rubygems_module.NewMarshalEncoder(zw).Encode(spec); err != nil {
		ctx.ServerError("Download file failed", err)
	}
}

// DownloadPackageFile serves the content of a package
func DownloadPackageFile(ctx *context.Context) {
	filename := ctx.Params("filename")

	pvs, err := getVersionsByFilename(ctx, filename)
	if err != nil {
		apiError(ctx, http.StatusInternalServerError, err)
		return
	}

	if len(pvs) != 1 {
		apiError(ctx, http.StatusNotFound, nil)
		return
	}

	s, u, pf, err := packages_service.GetFileStreamByPackageVersion(
		ctx,
		pvs[0],
		&packages_service.PackageFileInfo{
			Filename: filename,
		},
	)
	if err != nil {
		if err == packages_model.ErrPackageFileNotExist {
			apiError(ctx, http.StatusNotFound, err)
			return
		}
		apiError(ctx, http.StatusInternalServerError, err)
		return
	}

	helper.ServePackageFile(ctx, s, u, pf)
}

// UploadPackageFile adds a file to the package. If the package does not exist, it gets created.
func UploadPackageFile(ctx *context.Context) {
	upload, needToClose, err := ctx.UploadStream()
	if err != nil {
		apiError(ctx, http.StatusBadRequest, err)
		return
	}
	if needToClose {
		defer upload.Close()
	}

	buf, err := packages_module.CreateHashedBufferFromReader(upload)
	if err != nil {
		apiError(ctx, http.StatusInternalServerError, err)
		return
	}
	defer buf.Close()

	rp, err := rubygems_module.ParsePackageMetaData(buf)
	if err != nil {
		if errors.Is(err, util.ErrInvalidArgument) {
			apiError(ctx, http.StatusBadRequest, err)
		} else {
			apiError(ctx, http.StatusInternalServerError, err)
		}
		return
	}
	if _, err := buf.Seek(0, io.SeekStart); err != nil {
		apiError(ctx, http.StatusInternalServerError, err)
		return
	}

	filename := getFullFilename(rp.Name, rp.Version, rp.Metadata.Platform)

	_, _, err = packages_service.CreatePackageAndAddFile(
		ctx,
		&packages_service.PackageCreationInfo{
			PackageInfo: packages_service.PackageInfo{
				Owner:       ctx.Package.Owner,
				PackageType: packages_model.TypeRubyGems,
				Name:        rp.Name,
				Version:     rp.Version,
			},
			SemverCompatible: true,
			Creator:          ctx.Doer,
			Metadata:         rp.Metadata,
		},
		&packages_service.PackageFileCreationInfo{
			PackageFileInfo: packages_service.PackageFileInfo{
				Filename: filename,
			},
			Creator: ctx.Doer,
			Data:    buf,
			IsLead:  true,
		},
	)
	if err != nil {
		switch err {
		case packages_model.ErrDuplicatePackageVersion:
			apiError(ctx, http.StatusConflict, err)
		case packages_service.ErrQuotaTotalCount, packages_service.ErrQuotaTypeSize, packages_service.ErrQuotaTotalSize:
			apiError(ctx, http.StatusForbidden, err)
		default:
			apiError(ctx, http.StatusInternalServerError, err)
		}
		return
	}

	ctx.Status(http.StatusCreated)
}

// DeletePackage deletes a package
func DeletePackage(ctx *context.Context) {
	// Go populates the form only for POST, PUT and PATCH requests
	if err := ctx.Req.ParseMultipartForm(32 << 20); err != nil {
		apiError(ctx, http.StatusInternalServerError, err)
		return
	}
	packageName := ctx.FormString("gem_name")
	packageVersion := ctx.FormString("version")

	err := packages_service.RemovePackageVersionByNameAndVersion(
		ctx,
		ctx.Doer,
		&packages_service.PackageInfo{
			Owner:       ctx.Package.Owner,
			PackageType: packages_model.TypeRubyGems,
			Name:        packageName,
			Version:     packageVersion,
		},
	)
	if err != nil {
		if err == packages_model.ErrPackageNotExist {
			apiError(ctx, http.StatusNotFound, err)
			return
		}
		apiError(ctx, http.StatusInternalServerError, err)
	}
}

func writeRequirements(reqs []rubygems_module.VersionRequirement, result *strings.Builder) {
	if len(reqs) == 0 {
		reqs = []rubygems_module.VersionRequirement{{Restriction: ">=", Version: "0"}}
	}
	for i, req := range reqs {
		if i != 0 {
			result.WriteString("&")
		}
		result.WriteString(req.Restriction)
		result.WriteString(" ")
		result.WriteString(req.Version)
	}
}

func buildRequirementStringFromVersion(ctx *context.Context, version *packages_model.PackageVersion) (string, error) {
	pd, err := packages_model.GetPackageDescriptor(ctx, version)
	if err != nil {
		return "", err
	}
	metadata := pd.Metadata.(*rubygems_module.Metadata)
	dependencyRequirements := new(strings.Builder)
	for i, dep := range metadata.RuntimeDependencies {
		if i != 0 {
			dependencyRequirements.WriteString(",")
		}

		dependencyRequirements.WriteString(dep.Name)
		dependencyRequirements.WriteString(":")
		reqs := dep.Version
		writeRequirements(reqs, dependencyRequirements)
	}
	fullname := getFullFilename(pd.Package.Name, version.Version, metadata.Platform)
	file, err := packages_model.GetFileForVersionByName(ctx, version.ID, fullname, "")
	if err != nil {
		return "", err
	}
	blob, err := packages_model.GetBlobByID(ctx, file.BlobID)
	if err != nil {
		return "", err
	}
	additionalRequirements := new(strings.Builder)
	fmt.Fprintf(additionalRequirements, "checksum:%s", blob.HashSHA256)
	if len(metadata.RequiredRubyVersion) != 0 {
		additionalRequirements.WriteString(",ruby:")
		writeRequirements(metadata.RequiredRubyVersion, additionalRequirements)
	}
	if len(metadata.RequiredRubygemsVersion) != 0 {
		additionalRequirements.WriteString(",rubygems:")
		writeRequirements(metadata.RequiredRubygemsVersion, additionalRequirements)
	}

	if metadata.Platform != "ruby" {
		return fmt.Sprintf("%s-%s %s|%s", version.Version, metadata.Platform, dependencyRequirements, additionalRequirements), nil
	}

	return fmt.Sprintf("%s %s|%s", version.Version, dependencyRequirements, additionalRequirements), nil
}

func buildInfoFileForPackage(ctx *context.Context, versions []*packages_model.PackageVersion) (*string, error) {
	result := "---\n"
	for _, v := range versions {
		str, err := buildRequirementStringFromVersion(ctx, v)
		if err != nil {
			return nil, err
		}
		result += str
		result += "\n"
	}
	return &result, nil
}

func getFullFilename(gemName, version, platform string) string {
	return strings.ToLower(getFullName(gemName, version, platform)) + ".gem"
}

func getFullName(gemName, version, platform string) string {
	if platform == "" || platform == "ruby" {
		return fmt.Sprintf("%s-%s", gemName, version)
	}
	return fmt.Sprintf("%s-%s-%s", gemName, version, platform)
}

func getVersionsByFilename(ctx *context.Context, filename string) ([]*packages_model.PackageVersion, error) {
	pvs, _, err := packages_model.SearchVersions(ctx, &packages_model.PackageSearchOptions{
		OwnerID:         ctx.Package.Owner.ID,
		Type:            packages_model.TypeRubyGems,
		HasFileWithName: filename,
		IsInternal:      optional.Some(false),
	})
	return pvs, err
}
