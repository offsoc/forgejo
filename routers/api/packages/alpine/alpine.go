// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package alpine

import (
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"strings"

	packages_model "code.gitea.io/gitea/models/packages"
	"code.gitea.io/gitea/modules/util"
	"code.gitea.io/gitea/routers/api/packages/helper"
	"code.gitea.io/gitea/services/context"
	packages_service "code.gitea.io/gitea/services/packages"
	alpine_packages_service "code.gitea.io/gitea/services/packages/alpine"
)

func GetRepositoryKey(ctx *context.Context) {
	_, pub, err := alpine_packages_service.GetOrCreateKeyPair(ctx, ctx.Package.Owner.ID)
	if err != nil {
		helper.ApiError(ctx, http.StatusInternalServerError, err)
		return
	}

	pubPem, _ := pem.Decode([]byte(pub))
	if pubPem == nil {
		helper.ApiError(ctx, http.StatusInternalServerError, "failed to decode private key pem")
		return
	}

	pubKey, err := x509.ParsePKIXPublicKey(pubPem.Bytes)
	if err != nil {
		helper.ApiError(ctx, http.StatusInternalServerError, err)
		return
	}

	fingerprint, err := util.CreatePublicKeyFingerprint(pubKey)
	if err != nil {
		helper.ApiError(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.ServeContent(strings.NewReader(pub), &context.ServeHeaderOptions{
		ContentType: "application/x-pem-file",
		Filename:    fmt.Sprintf("%s@%s.rsa.pub", ctx.Package.Owner.LowerName, hex.EncodeToString(fingerprint)),
	})
}

func GetRepositoryFile(ctx *context.Context) {
	pv, err := alpine_packages_service.GetOrCreateRepositoryVersion(ctx, ctx.Package.Owner.ID)
	if err != nil {
		helper.ApiError(ctx, http.StatusInternalServerError, err)
		return
	}

	s, u, pf, err := packages_service.GetFileStreamByPackageVersion(
		ctx,
		pv,
		&packages_service.PackageFileInfo{
			Filename:     alpine_packages_service.IndexArchiveFilename,
			CompositeKey: fmt.Sprintf("%s|%s|%s", ctx.Params("branch"), ctx.Params("repository"), ctx.Params("architecture")),
		},
	)
	if err != nil {
		if errors.Is(err, util.ErrNotExist) {
			helper.ApiError(ctx, http.StatusNotFound, err)
		} else {
			helper.ApiError(ctx, http.StatusInternalServerError, err)
		}
		return
	}

	helper.ServePackageFile(ctx, s, u, pf)
}

func UploadPackageFile(ctx *context.Context) {
	branch := strings.TrimSpace(ctx.Params("branch"))
	repository := strings.TrimSpace(ctx.Params("repository"))
	if branch == "" || repository == "" {
		helper.ApiError(ctx, http.StatusBadRequest, "invalid branch or repository")
		return
	}

	upload, needToClose, err := ctx.UploadStream()
	if err != nil {
		helper.ApiError(ctx, http.StatusInternalServerError, err)
		return
	}
	if needToClose {
		defer upload.Close()
	}

	_, err = alpine_packages_service.UploadPackage(ctx, branch, repository, upload, ctx.Package.Owner, ctx.Doer)
	if err != nil {
		helper.PackageUploadError(ctx, err)
		return
	}

	ctx.Status(http.StatusCreated)
}

func DownloadPackageFile(ctx *context.Context) {
	branch := ctx.Params("branch")
	repository := ctx.Params("repository")
	architecture := ctx.Params("architecture")

	opts := &packages_model.PackageFileSearchOptions{
		OwnerID:      ctx.Package.Owner.ID,
		PackageType:  packages_model.TypeAlpine,
		Query:        ctx.Params("filename"),
		CompositeKey: fmt.Sprintf("%s|%s|%s", branch, repository, architecture),
	}

	pfs, _, err := packages_model.SearchFiles(ctx, opts)
	if err != nil {
		helper.ApiError(ctx, http.StatusInternalServerError, err)
		return
	}
	if len(pfs) == 0 {
		helper.ApiError(ctx, http.StatusNotFound, nil)
		return
	}

	s, u, pf, err := packages_service.GetPackageFileStream(ctx, pfs[0])
	if err != nil {
		if errors.Is(err, util.ErrNotExist) {
			helper.ApiError(ctx, http.StatusNotFound, err)
		} else {
			helper.ApiError(ctx, http.StatusInternalServerError, err)
		}
		return
	}

	helper.ServePackageFile(ctx, s, u, pf)
}

func DeletePackageFile(ctx *context.Context) {
	branch, repository, architecture := ctx.Params("branch"), ctx.Params("repository"), ctx.Params("architecture")

	pfs, _, err := packages_model.SearchFiles(ctx, &packages_model.PackageFileSearchOptions{
		OwnerID:      ctx.Package.Owner.ID,
		PackageType:  packages_model.TypeAlpine,
		Query:        ctx.Params("filename"),
		CompositeKey: fmt.Sprintf("%s|%s|%s", branch, repository, architecture),
	})
	if err != nil {
		helper.ApiError(ctx, http.StatusInternalServerError, err)
		return
	}
	if len(pfs) != 1 {
		helper.ApiError(ctx, http.StatusNotFound, nil)
		return
	}

	if err := packages_service.RemovePackageFileAndVersionIfUnreferenced(ctx, ctx.Doer, pfs[0]); err != nil {
		if errors.Is(err, util.ErrNotExist) {
			helper.ApiError(ctx, http.StatusNotFound, err)
		} else {
			helper.ApiError(ctx, http.StatusInternalServerError, err)
		}
		return
	}

	if err := alpine_packages_service.BuildSpecificRepositoryFiles(ctx, ctx.Package.Owner.ID, branch, repository, architecture); err != nil {
		helper.ApiError(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.Status(http.StatusNoContent)
}
