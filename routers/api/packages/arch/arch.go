// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package arch

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"

	packages_model "forgejo.org/models/packages"
	packages_module "forgejo.org/modules/packages"
	arch_module "forgejo.org/modules/packages/arch"
	"forgejo.org/modules/sync"
	"forgejo.org/modules/util"
	"forgejo.org/routers/api/packages/helper"
	"forgejo.org/services/context"
	packages_service "forgejo.org/services/packages"
	arch_service "forgejo.org/services/packages/arch"
)

var (
	archPkgOrSig = regexp.MustCompile(`^.*\.pkg\.tar\.\w+(\.sig)*$`)
	archDBOrSig  = regexp.MustCompile(`^.*.(db|files)(\.tar\.gz)*(\.sig)*$`)

	locker = sync.NewExclusivePool()
)

func apiError(ctx *context.Context, status int, obj any) {
	helper.LogAndProcessError(ctx, status, obj, func(message string) {
		ctx.PlainText(status, message)
	})
}

func refreshLocker(ctx *context.Context, group string) func() {
	key := fmt.Sprintf("pkg_%d_arch_pkg_%s", ctx.Package.Owner.ID, group)
	locker.CheckIn(key)
	return func() {
		locker.CheckOut(key)
	}
}

func GetRepositoryKey(ctx *context.Context) {
	_, pub, err := arch_service.GetOrCreateKeyPair(ctx, ctx.Package.Owner.ID)
	if err != nil {
		apiError(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.ServeContent(strings.NewReader(pub), &context.ServeHeaderOptions{
		ContentType: "application/pgp-keys",
		Filename:    "repository.key",
	})
}

func PushPackage(ctx *context.Context) {
	group := strings.Trim(ctx.Params("*"), "/")
	releaser := refreshLocker(ctx, group)
	defer releaser()
	upload, needToClose, err := ctx.UploadStream()
	if err != nil {
		apiError(ctx, http.StatusInternalServerError, err)
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

	p, err := arch_module.ParsePackage(buf)
	if err != nil {
		apiError(ctx, http.StatusBadRequest, err)
		return
	}

	_, err = buf.Seek(0, io.SeekStart)
	if err != nil {
		apiError(ctx, http.StatusInternalServerError, err)
		return
	}
	sign, err := arch_service.NewFileSign(ctx, ctx.Package.Owner.ID, buf)
	if err != nil {
		apiError(ctx, http.StatusInternalServerError, err)
		return
	}
	defer sign.Close()
	_, err = buf.Seek(0, io.SeekStart)
	if err != nil {
		apiError(ctx, http.StatusInternalServerError, err)
		return
	}
	// update gpg sign
	pgp, err := io.ReadAll(sign)
	if err != nil {
		apiError(ctx, http.StatusInternalServerError, err)
		return
	}
	p.FileMetadata.PgpSigned = base64.StdEncoding.EncodeToString(pgp)
	_, err = sign.Seek(0, io.SeekStart)
	if err != nil {
		apiError(ctx, http.StatusInternalServerError, err)
		return
	}

	properties := map[string]string{
		arch_module.PropertyDescription:  p.Desc(),
		arch_module.PropertyFiles:        p.Files(),
		arch_module.PropertyArch:         p.FileMetadata.Arch,
		arch_module.PropertyDistribution: group,
	}

	version, _, err := packages_service.CreatePackageOrAddFileToExisting(
		ctx,
		&packages_service.PackageCreationInfo{
			PackageInfo: packages_service.PackageInfo{
				Owner:       ctx.Package.Owner,
				PackageType: packages_model.TypeArch,
				Name:        p.Name,
				Version:     p.Version,
			},
			Creator:  ctx.Doer,
			Metadata: p.VersionMetadata,
		},
		&packages_service.PackageFileCreationInfo{
			PackageFileInfo: packages_service.PackageFileInfo{
				Filename:     fmt.Sprintf("%s-%s-%s.pkg.tar.%s", p.Name, p.Version, p.FileMetadata.Arch, p.CompressType),
				CompositeKey: group,
			},
			OverwriteExisting: false,
			IsLead:            true,
			Creator:           ctx.ContextUser,
			Data:              buf,
			Properties:        properties,
		},
	)
	if err != nil {
		switch {
		case errors.Is(err, packages_model.ErrDuplicatePackageVersion), errors.Is(err, packages_model.ErrDuplicatePackageFile):
			apiError(ctx, http.StatusConflict, err)
		case errors.Is(err, packages_service.ErrQuotaTotalCount), errors.Is(err, packages_service.ErrQuotaTypeSize), errors.Is(err, packages_service.ErrQuotaTotalSize):
			apiError(ctx, http.StatusForbidden, err)
		default:
			apiError(ctx, http.StatusInternalServerError, err)
		}
		return
	}
	// add sign file
	_, err = packages_service.AddFileToPackageVersionInternal(ctx, version, &packages_service.PackageFileCreationInfo{
		PackageFileInfo: packages_service.PackageFileInfo{
			CompositeKey: group,
			Filename:     fmt.Sprintf("%s-%s-%s.pkg.tar.%s.sig", p.Name, p.Version, p.FileMetadata.Arch, p.CompressType),
		},
		OverwriteExisting: true,
		IsLead:            false,
		Creator:           ctx.Doer,
		Data:              sign,
	})
	if err != nil {
		apiError(ctx, http.StatusInternalServerError, err)
		return
	}
	if err = arch_service.BuildPacmanDB(ctx, ctx.Package.Owner.ID, group, p.FileMetadata.Arch); err != nil {
		apiError(ctx, http.StatusInternalServerError, err)
		return
	}
	if p.FileMetadata.Arch == "any" {
		if err = arch_service.BuildCustomRepositoryFiles(ctx, ctx.Package.Owner.ID, group); err != nil {
			apiError(ctx, http.StatusInternalServerError, err)
			return
		}
	}
	ctx.Status(http.StatusCreated)
}

func GetPackageOrDB(ctx *context.Context) {
	pathGroups := strings.Split(strings.Trim(ctx.Params("*"), "/"), "/")
	groupLen := len(pathGroups)
	if groupLen < 2 {
		ctx.Status(http.StatusNotFound)
		return
	}
	var file, group, arch string
	if groupLen == 2 {
		arch = pathGroups[0]
		file = pathGroups[1]
	} else {
		group = strings.Join(pathGroups[:groupLen-2], "/")
		arch = pathGroups[groupLen-2]
		file = pathGroups[groupLen-1]
	}
	if archPkgOrSig.MatchString(file) {
		pkg, u, pf, err := arch_service.GetPackageFile(ctx, group, file, ctx.Package.Owner.ID)
		if err != nil {
			if errors.Is(err, util.ErrNotExist) {
				apiError(ctx, http.StatusNotFound, err)
			} else {
				apiError(ctx, http.StatusInternalServerError, err)
			}
			return
		}
		helper.ServePackageFile(ctx, pkg, u, pf)
		return
	}

	if archDBOrSig.MatchString(file) {
		pkg, u, pf, err := arch_service.GetPackageDBFile(ctx, ctx.Package.Owner.ID, group, arch, strings.HasSuffix(file, ".sig"))
		if err != nil {
			if errors.Is(err, util.ErrNotExist) {
				apiError(ctx, http.StatusNotFound, err)
			} else {
				apiError(ctx, http.StatusInternalServerError, err)
			}
			return
		}
		helper.ServePackageFile(ctx, pkg, u, pf)
		return
	}

	ctx.Status(http.StatusNotFound)
}

func RemovePackage(ctx *context.Context) {
	pathGroups := strings.Split(strings.Trim(ctx.Params("*"), "/"), "/")
	groupLen := len(pathGroups)
	if groupLen < 3 {
		ctx.Status(http.StatusBadRequest)
		return
	}
	var group, pkg, ver, pkgArch string
	if groupLen == 3 {
		pkg = pathGroups[0]
		ver = pathGroups[1]
		pkgArch = pathGroups[2]
	} else {
		group = strings.Join(pathGroups[:groupLen-3], "/")
		pkg = pathGroups[groupLen-3]
		ver = pathGroups[groupLen-2]
		pkgArch = pathGroups[groupLen-1]
	}
	releaser := refreshLocker(ctx, group)
	defer releaser()
	pv, err := packages_model.GetVersionByNameAndVersion(
		ctx, ctx.Package.Owner.ID, packages_model.TypeArch, pkg, ver,
	)
	if err != nil {
		if errors.Is(err, util.ErrNotExist) {
			apiError(ctx, http.StatusNotFound, err)
		} else {
			apiError(ctx, http.StatusInternalServerError, err)
		}
		return
	}
	files, err := packages_model.GetFilesByVersionID(ctx, pv.ID)
	if err != nil {
		apiError(ctx, http.StatusInternalServerError, err)
		return
	}
	deleted := false
	for _, file := range files {
		extName := fmt.Sprintf("-%s.pkg.tar%s", pkgArch, filepath.Ext(file.LowerName))
		if strings.HasSuffix(file.LowerName, ".sig") {
			extName = fmt.Sprintf("-%s.pkg.tar%s.sig", pkgArch,
				filepath.Ext(strings.TrimSuffix(file.LowerName, filepath.Ext(file.LowerName))))
		}
		if file.CompositeKey == group &&
			strings.HasSuffix(file.LowerName, extName) {
			deleted = true
			err := packages_service.RemovePackageFileAndVersionIfUnreferenced(ctx, ctx.ContextUser, file)
			if err != nil {
				apiError(ctx, http.StatusInternalServerError, err)
				return
			}
		}
	}
	if deleted {
		err = arch_service.BuildCustomRepositoryFiles(ctx, ctx.Package.Owner.ID, group)
		if err != nil {
			apiError(ctx, http.StatusInternalServerError, err)
			return
		}
		ctx.Status(http.StatusNoContent)
	} else {
		ctx.Error(http.StatusNotFound)
	}
}
