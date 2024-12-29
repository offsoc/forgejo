// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package packages

import (
	"errors"
	"fmt"
	"io"

	packages_model "code.gitea.io/gitea/models/packages"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/modules/util"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/services/context"
	"code.gitea.io/gitea/services/forms"
	packages_service "code.gitea.io/gitea/services/packages"
	alpine_packages_service "code.gitea.io/gitea/services/packages/alpine"
	debian_packages_service "code.gitea.io/gitea/services/packages/debian"
	generic_packages_service "code.gitea.io/gitea/services/packages/generic"
	rpm_packages_service "code.gitea.io/gitea/services/packages/rpm"
)

var UploadTypeList = []packages_model.Type{
	packages_model.TypeAlpine,
	packages_model.TypeDebian,
	packages_model.TypeGeneric,
	packages_model.TypeRpm,
}

func servePackageUploadUserError(ctx *context.Context, err error, packageType, repo string) {
	ctx.Flash.Error(err.Error())

	if repo == "" {
		ctx.Redirect(fmt.Sprintf("%s/-/packages/upload/%s", ctx.ContextUser.HTMLURL(), packageType))
	} else {
		ctx.Redirect(fmt.Sprintf("%s/-/packages/upload/%s?repo=%s", ctx.ContextUser.HTMLURL(), packageType, repo))
	}
}

func servePackageUploadError(ctx *context.Context, err error, packageType, repo string) {
	userErroes := []error{
		packages_service.ErrQuotaTotalCount,
		packages_service.ErrQuotaTypeSize,
		packages_service.ErrQuotaTotalSize,
		packages_model.ErrDuplicatePackageVersion,
		packages_model.ErrDuplicatePackageFile,
		util.ErrInvalidArgument,
		io.EOF,
	}

	for _, currentError := range userErroes {
		if errors.Is(err, currentError) {
			servePackageUploadUserError(ctx, err, packageType, repo)
			return
		}
	}

	ctx.ServerError(fmt.Sprintf("Upload %s package", packageType), err)
}

func addRepoToUploadedPackage(ctx *context.Context, packageType, repoName string, packageID int64) bool {
	repo, err := repo_model.GetRepositoryByOwnerAndName(ctx, ctx.ContextUser.Name, repoName)
	if err != nil {
		if repo_model.IsErrRepoNotExist(err) {
			servePackageUploadError(ctx, fmt.Errorf("repo not found"), packageType, repoName)
			return false
		}

		ctx.ServerError("GetRepositoryByOwnerAndName", err)
		return false
	}

	err = packages_model.SetRepositoryLink(ctx, packageID, repo.ID)
	if err != nil {
		ctx.ServerError("SetRepositoryLink", err)
		return false
	}

	return true
}

func uploadPackageFinish(ctx *context.Context, packageType, packageRepo string, pv *packages_model.PackageVersion) {
	if packageRepo != "" {
		if !addRepoToUploadedPackage(ctx, packageType, packageRepo, pv.PackageID) {
			return
		}
	}

	pd, err := packages_model.GetPackageDescriptor(ctx, pv)
	if err != nil {
		ctx.ServerError("GetPackageDescriptor", err)
		return
	}

	ctx.Redirect(pd.PackageWebLink())
}

func UploadAlpinePackagePost(ctx *context.Context) {
	form := web.GetForm(ctx).(*forms.PackageUploadAlpineForm)
	upload, err := form.File.Open()
	if err != nil {
		ctx.ServerError("GetPackageFile", err)
		return
	}
	defer upload.Close()

	pv, err := alpine_packages_service.UploadPackage(ctx, form.Branch, form.Repository, upload, ctx.Package.Owner, ctx.Doer)
	if err != nil {
		servePackageUploadError(ctx, err, "alpine", form.Repo)
		return
	}

	uploadPackageFinish(ctx, "alpine", form.Repo, pv)
}

func UploadDebianPackagePost(ctx *context.Context) {
	form := web.GetForm(ctx).(*forms.PackageUploadDebianForm)
	upload, err := form.File.Open()
	if err != nil {
		ctx.ServerError("GetPackageFile", err)
		return
	}
	defer upload.Close()

	pv, err := debian_packages_service.UploadPackage(ctx, form.Distribution, form.Component, upload, ctx.Package.Owner, ctx.Doer)
	if err != nil {
		servePackageUploadError(ctx, err, "debian", form.Repo)
		return
	}

	uploadPackageFinish(ctx, "debian", form.Repo, pv)
}

func UploadGenericPackagePost(ctx *context.Context) {
	form := web.GetForm(ctx).(*forms.PackageUploadGenericForm)
	upload, err := form.File.Open()
	if err != nil {
		ctx.ServerError("GetPackageFile", err)
		return
	}
	defer upload.Close()

	var filename string
	if form.Filename == "" {
		filename = form.File.Filename
	} else {
		filename = form.Filename
	}

	pv, err := generic_packages_service.UploadPackage(ctx, form.Name, form.Version, filename, upload, ctx.Package.Owner, ctx.Doer)
	if err != nil {
		servePackageUploadError(ctx, err, "generic", form.Repo)
		return
	}

	uploadPackageFinish(ctx, "generic", form.Repo, pv)
}

func UploadRpmPackagePost(ctx *context.Context) {
	form := web.GetForm(ctx).(*forms.PackageUploadRpmForm)
	upload, err := form.File.Open()
	if err != nil {
		ctx.ServerError("GetPackageFile", err)
		return
	}
	defer upload.Close()

	pv, err := rpm_packages_service.UploadPackage(ctx, form.Sign, form.Group, upload, ctx.Package.Owner, ctx.Doer)
	if err != nil {
		servePackageUploadError(ctx, err, "rpm", form.Repo)
		return
	}

	uploadPackageFinish(ctx, "rpm", form.Repo, pv)
}
