// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package maven

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	packages_model "forgejo.org/models/packages"
	"forgejo.org/modules/json"
	"forgejo.org/modules/log"
	packages_module "forgejo.org/modules/packages"
	maven_module "forgejo.org/modules/packages/maven"
	"forgejo.org/modules/sync"
	"forgejo.org/routers/api/packages/helper"
	"forgejo.org/services/context"
	packages_service "forgejo.org/services/packages"
)

const (
	mavenMetadataFile = "maven-metadata.xml"
	extensionMD5      = ".md5"
	extensionSHA1     = ".sha1"
	extensionSHA256   = ".sha256"
	extensionSHA512   = ".sha512"
	extensionPom      = ".pom"
	extensionJar      = ".jar"
	contentTypeJar    = "application/java-archive"
	contentTypeXML    = "text/xml"
)

var (
	errInvalidParameters = errors.New("request parameters are invalid")
	illegalCharacters    = regexp.MustCompile(`[\\/:"<>|?\*]`)
)

func apiError(ctx *context.Context, status int, obj any) {
	helper.LogAndProcessError(ctx, status, obj, func(message string) {
		// The maven client does not present the error message to the user. Log it for users with access to server logs.
		switch status {
		case http.StatusBadRequest:
			log.Warn(message)
		case http.StatusInternalServerError:
			log.Error(message)
		}

		ctx.PlainText(status, message)
	})
}

// buildPackageID creates a package ID from group and artifact ID
// Refer to https://maven.apache.org/pom.html#Maven_Coordinates
func buildPackageID(groupID, artifactID string) string {
	return fmt.Sprintf("%s:%s", groupID, artifactID)
}

// DownloadPackageFile serves the content of a package
func DownloadPackageFile(ctx *context.Context) {
	handlePackageFile(ctx, true)
}

// ProvidePackageFileHeader provides only the headers describing a package
func ProvidePackageFileHeader(ctx *context.Context) {
	handlePackageFile(ctx, false)
}

func handlePackageFile(ctx *context.Context, serveContent bool) {
	params, err := extractPathParameters(ctx)
	if err != nil {
		apiError(ctx, http.StatusBadRequest, err)
		return
	}

	if params.IsMeta && params.Version == "" {
		serveMavenMetadata(ctx, params)
	} else {
		servePackageFile(ctx, params, serveContent)
	}
}

func serveMavenMetadata(ctx *context.Context, params parameters) {
	// /com/foo/project/maven-metadata.xml[.md5/.sha1/.sha256/.sha512]

	packageName := buildPackageID(params.GroupID, params.ArtifactID)
	pvs, err := packages_model.GetVersionsByPackageName(ctx, ctx.Package.Owner.ID, packages_model.TypeMaven, packageName)
	if err != nil {
		apiError(ctx, http.StatusInternalServerError, err)
		return
	}
	if len(pvs) == 0 {
		apiError(ctx, http.StatusNotFound, packages_model.ErrPackageNotExist)
		return
	}

	pds, err := packages_model.GetPackageDescriptors(ctx, pvs)
	if err != nil {
		apiError(ctx, http.StatusInternalServerError, err)
		return
	}

	sort.Slice(pds, func(i, j int) bool {
		// Maven and Gradle order packages by their creation timestamp and not by their version string
		return pds[i].Version.CreatedUnix < pds[j].Version.CreatedUnix
	})

	xmlMetadata, err := xml.Marshal(createMetadataResponse(pds))
	if err != nil {
		apiError(ctx, http.StatusInternalServerError, err)
		return
	}
	xmlMetadataWithHeader := append([]byte(xml.Header), xmlMetadata...)

	latest := pds[len(pds)-1]
	// http.TimeFormat required a UTC time, refer to https://pkg.go.dev/net/http#TimeFormat
	lastModified := latest.Version.CreatedUnix.AsTime().UTC().Format(http.TimeFormat)
	ctx.Resp.Header().Set("Last-Modified", lastModified)

	ext := strings.ToLower(filepath.Ext(params.Filename))
	if isChecksumExtension(ext) {
		var hash []byte
		switch ext {
		case extensionMD5:
			tmp := md5.Sum(xmlMetadataWithHeader)
			hash = tmp[:]
		case extensionSHA1:
			tmp := sha1.Sum(xmlMetadataWithHeader)
			hash = tmp[:]
		case extensionSHA256:
			tmp := sha256.Sum256(xmlMetadataWithHeader)
			hash = tmp[:]
		case extensionSHA512:
			tmp := sha512.Sum512(xmlMetadataWithHeader)
			hash = tmp[:]
		}
		ctx.PlainText(http.StatusOK, hex.EncodeToString(hash))
		return
	}

	ctx.Resp.Header().Set("Content-Length", strconv.Itoa(len(xmlMetadataWithHeader)))
	ctx.Resp.Header().Set("Content-Type", contentTypeXML)

	_, _ = ctx.Resp.Write(xmlMetadataWithHeader)
}

func servePackageFile(ctx *context.Context, params parameters, serveContent bool) {
	packageName := buildPackageID(params.GroupID, params.ArtifactID)

	pv, err := packages_model.GetVersionByNameAndVersion(ctx, ctx.Package.Owner.ID, packages_model.TypeMaven, packageName, params.Version)
	if err != nil {
		if err == packages_model.ErrPackageNotExist {
			apiError(ctx, http.StatusNotFound, err)
		} else {
			apiError(ctx, http.StatusInternalServerError, err)
		}
		return
	}

	filename := params.Filename

	ext := strings.ToLower(filepath.Ext(filename))
	if isChecksumExtension(ext) {
		filename = filename[:len(filename)-len(ext)]
	}

	pf, err := packages_model.GetFileForVersionByNameMatchCase(ctx, pv.ID, filename, packages_model.EmptyFileKey)
	if err != nil {
		if errors.Is(err, packages_model.ErrPackageFileNotExist) {
			apiError(ctx, http.StatusNotFound, err)
		} else {
			apiError(ctx, http.StatusInternalServerError, err)
		}
		return
	}

	pb, err := packages_model.GetBlobByID(ctx, pf.BlobID)
	if err != nil {
		apiError(ctx, http.StatusInternalServerError, err)
		return
	}

	if isChecksumExtension(ext) {
		var hash string
		switch ext {
		case extensionMD5:
			hash = pb.HashMD5
		case extensionSHA1:
			hash = pb.HashSHA1
		case extensionSHA256:
			hash = pb.HashSHA256
		case extensionSHA512:
			hash = pb.HashSHA512
		}
		ctx.PlainText(http.StatusOK, hash)
		return
	}

	opts := &context.ServeHeaderOptions{
		ContentLength: &pb.Size,
		LastModified:  pf.CreatedUnix.AsLocalTime(),
	}
	switch ext {
	case extensionJar:
		opts.ContentType = contentTypeJar
	case extensionPom:
		opts.ContentType = contentTypeXML
	}

	if !serveContent {
		ctx.SetServeHeaders(opts)
		ctx.Status(http.StatusOK)
		return
	}

	s, u, _, err := packages_service.GetPackageBlobStream(ctx, pf, pb, nil)
	if err != nil {
		apiError(ctx, http.StatusInternalServerError, err)
		return
	}

	opts.Filename = pf.Name

	helper.ServePackageFile(ctx, s, u, pf, opts)
}

var mavenUploadLock = sync.NewExclusivePool()

// UploadPackageFile adds a file to the package. If the package does not exist, it gets created.
func UploadPackageFile(ctx *context.Context) {
	params, err := extractPathParameters(ctx)
	if err != nil {
		apiError(ctx, http.StatusBadRequest, err)
		return
	}

	log.Trace("Parameters: %+v", params)

	// Ignore the package index /<name>/maven-metadata.xml
	if params.IsMeta && params.Version == "" {
		ctx.Status(http.StatusOK)
		return
	}

	packageName := buildPackageID(params.GroupID, params.ArtifactID)

	mavenUploadLock.CheckIn(packageName)
	defer mavenUploadLock.CheckOut(packageName)

	buf, err := packages_module.CreateHashedBufferFromReader(ctx.Req.Body)
	if err != nil {
		apiError(ctx, http.StatusInternalServerError, err)
		return
	}
	defer buf.Close()

	pvci := &packages_service.PackageCreationInfo{
		PackageInfo: packages_service.PackageInfo{
			Owner:       ctx.Package.Owner,
			PackageType: packages_model.TypeMaven,
			Name:        packageName,
			Version:     params.Version,
		},
		SemverCompatible: false,
		Creator:          ctx.Doer,
	}

	ext := filepath.Ext(params.Filename)

	// Do not upload checksum files but compare the hashes.
	if isChecksumExtension(ext) {
		pv, err := packages_model.GetVersionByNameAndVersion(ctx, pvci.Owner.ID, pvci.PackageType, pvci.Name, pvci.Version)
		if err != nil {
			if err == packages_model.ErrPackageNotExist {
				apiError(ctx, http.StatusNotFound, err)
				return
			}
			apiError(ctx, http.StatusInternalServerError, err)
			return
		}
		pf, err := packages_model.GetFileForVersionByNameMatchCase(ctx, pv.ID, params.Filename[:len(params.Filename)-len(ext)], packages_model.EmptyFileKey)
		if err != nil {
			if errors.Is(err, packages_model.ErrPackageFileNotExist) {
				apiError(ctx, http.StatusNotFound, err)
				return
			}
			apiError(ctx, http.StatusInternalServerError, err)
			return
		}
		pb, err := packages_model.GetBlobByID(ctx, pf.BlobID)
		if err != nil {
			apiError(ctx, http.StatusInternalServerError, err)
			return
		}

		hash, err := io.ReadAll(buf)
		if err != nil {
			apiError(ctx, http.StatusInternalServerError, err)
			return
		}

		if (ext == extensionMD5 && pb.HashMD5 != string(hash)) ||
			(ext == extensionSHA1 && pb.HashSHA1 != string(hash)) ||
			(ext == extensionSHA256 && pb.HashSHA256 != string(hash)) ||
			(ext == extensionSHA512 && pb.HashSHA512 != string(hash)) {
			apiError(ctx, http.StatusBadRequest, "hash mismatch")
			return
		}

		ctx.Status(http.StatusOK)
		return
	}

	pfci := &packages_service.PackageFileCreationInfo{
		PackageFileInfo: packages_service.PackageFileInfo{
			Filename: params.Filename,
		},
		Creator:           ctx.Doer,
		Data:              buf,
		IsLead:            false,
		OverwriteExisting: params.IsMeta,
	}

	// If it's the package pom file extract the metadata
	if ext == extensionPom {
		pfci.IsLead = true

		var err error
		pvci.Metadata, err = maven_module.ParsePackageMetaData(buf)
		if err != nil {
			apiError(ctx, http.StatusBadRequest, err)
			return
		}

		if pvci.Metadata != nil {
			pv, err := packages_model.GetVersionByNameAndVersion(ctx, pvci.Owner.ID, pvci.PackageType, pvci.Name, pvci.Version)
			if err != nil && !errors.Is(err, packages_model.ErrPackageNotExist) {
				apiError(ctx, http.StatusInternalServerError, err)
				return
			}
			if pv != nil {
				raw, err := json.Marshal(pvci.Metadata)
				if err != nil {
					apiError(ctx, http.StatusInternalServerError, err)
					return
				}
				pv.MetadataJSON = string(raw)
				if err := packages_model.UpdateVersion(ctx, pv); err != nil {
					apiError(ctx, http.StatusInternalServerError, err)
					return
				}
			}
		}

		if _, err := buf.Seek(0, io.SeekStart); err != nil {
			apiError(ctx, http.StatusInternalServerError, err)
			return
		}
	}

	_, _, err = packages_service.CreatePackageOrAddFileToExisting(
		ctx,
		pvci,
		pfci,
	)
	if err != nil {
		switch err {
		case packages_model.ErrDuplicatePackageFile:
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

func isChecksumExtension(ext string) bool {
	return ext == extensionMD5 || ext == extensionSHA1 || ext == extensionSHA256 || ext == extensionSHA512
}

type parameters struct {
	GroupID    string
	ArtifactID string
	Version    string
	Filename   string
	IsMeta     bool
}

func extractPathParameters(ctx *context.Context) (parameters, error) {
	parts := strings.Split(ctx.Params("*"), "/")

	p := parameters{
		Filename: parts[len(parts)-1],
	}

	p.IsMeta = p.Filename == mavenMetadataFile ||
		p.Filename == mavenMetadataFile+extensionMD5 ||
		p.Filename == mavenMetadataFile+extensionSHA1 ||
		p.Filename == mavenMetadataFile+extensionSHA256 ||
		p.Filename == mavenMetadataFile+extensionSHA512

	parts = parts[:len(parts)-1]
	if len(parts) == 0 {
		return p, errInvalidParameters
	}

	p.Version = parts[len(parts)-1]
	if p.IsMeta && !strings.HasSuffix(p.Version, "-SNAPSHOT") {
		p.Version = ""
	} else {
		parts = parts[:len(parts)-1]
	}

	if illegalCharacters.MatchString(p.Version) {
		return p, errInvalidParameters
	}

	if len(parts) < 2 {
		return p, errInvalidParameters
	}

	p.ArtifactID = parts[len(parts)-1]
	p.GroupID = strings.Join(parts[:len(parts)-1], ".")

	if illegalCharacters.MatchString(p.GroupID) || illegalCharacters.MatchString(p.ArtifactID) {
		return p, errInvalidParameters
	}

	return p, nil
}
