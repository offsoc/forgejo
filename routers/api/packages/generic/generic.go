// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package generic

import (
	"net/http"

	packages_model "code.gitea.io/gitea/models/packages"
	"code.gitea.io/gitea/routers/api/packages/helper"
	"code.gitea.io/gitea/services/context"
	packages_service "code.gitea.io/gitea/services/packages"
	generic_packages_service "code.gitea.io/gitea/services/packages/generic"
)

// DownloadPackageFile serves the specific generic package.
func DownloadPackageFile(ctx *context.Context) {
	s, u, pf, err := packages_service.GetFileStreamByPackageNameAndVersion(
		ctx,
		&packages_service.PackageInfo{
			Owner:       ctx.Package.Owner,
			PackageType: packages_model.TypeGeneric,
			Name:        ctx.Params("packagename"),
			Version:     ctx.Params("packageversion"),
		},
		&packages_service.PackageFileInfo{
			Filename: ctx.Params("filename"),
		},
	)
	if err != nil {
		if err == packages_model.ErrPackageNotExist || err == packages_model.ErrPackageFileNotExist {
			helper.APIError(ctx, http.StatusNotFound, err)
			return
		}
		helper.APIError(ctx, http.StatusInternalServerError, err)
		return
	}

	helper.ServePackageFile(ctx, s, u, pf)
}

// UploadPackage uploads the specific generic package.
// Duplicated packages get rejected.
func UploadPackage(ctx *context.Context) {
	packageName := ctx.Params("packagename")
	filename := ctx.Params("filename")
	packageVersion := ctx.Params("packageversion")

	reader, needToClose, err := ctx.UploadStream()
	if err != nil {
		helper.APIError(ctx, http.StatusInternalServerError, err)
		return
	}
	if needToClose {
		defer reader.Close()
	}

	_, err = generic_packages_service.UploadPackage(ctx, packageName, filename, packageVersion, reader, ctx.Package.Owner, ctx.Doer)
	if err != nil {
		helper.PackageUploadError(ctx, err)
		return
	}

	ctx.Status(http.StatusCreated)
}

// DeletePackage deletes the specific generic package.
func DeletePackage(ctx *context.Context) {
	err := packages_service.RemovePackageVersionByNameAndVersion(
		ctx,
		ctx.Doer,
		&packages_service.PackageInfo{
			Owner:       ctx.Package.Owner,
			PackageType: packages_model.TypeGeneric,
			Name:        ctx.Params("packagename"),
			Version:     ctx.Params("packageversion"),
		},
	)
	if err != nil {
		if err == packages_model.ErrPackageNotExist {
			helper.APIError(ctx, http.StatusNotFound, err)
			return
		}
		helper.APIError(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.Status(http.StatusNoContent)
}

// DeletePackageFile deletes the specific file of a generic package.
func DeletePackageFile(ctx *context.Context) {
	pv, pf, err := func() (*packages_model.PackageVersion, *packages_model.PackageFile, error) {
		pv, err := packages_model.GetVersionByNameAndVersion(ctx, ctx.Package.Owner.ID, packages_model.TypeGeneric, ctx.Params("packagename"), ctx.Params("packageversion"))
		if err != nil {
			return nil, nil, err
		}

		pf, err := packages_model.GetFileForVersionByName(ctx, pv.ID, ctx.Params("filename"), packages_model.EmptyFileKey)
		if err != nil {
			return nil, nil, err
		}

		return pv, pf, nil
	}()
	if err != nil {
		if err == packages_model.ErrPackageNotExist || err == packages_model.ErrPackageFileNotExist {
			helper.APIError(ctx, http.StatusNotFound, err)
			return
		}
		helper.APIError(ctx, http.StatusInternalServerError, err)
		return
	}

	pfs, err := packages_model.GetFilesByVersionID(ctx, pv.ID)
	if err != nil {
		helper.APIError(ctx, http.StatusInternalServerError, err)
		return
	}

	if len(pfs) == 1 {
		if err := packages_service.RemovePackageVersion(ctx, ctx.Doer, pv); err != nil {
			helper.APIError(ctx, http.StatusInternalServerError, err)
			return
		}
	} else {
		if err := packages_service.DeletePackageFile(ctx, pf); err != nil {
			helper.APIError(ctx, http.StatusInternalServerError, err)
			return
		}
	}

	ctx.Status(http.StatusNoContent)
}
