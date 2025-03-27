// Copyright 2024 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"fmt"
	"net/http"

	"forgejo.org/modules/git"
	"forgejo.org/modules/gitrepo"
	"forgejo.org/services/context"
	archiver_service "forgejo.org/services/repository/archiver"
)

func DownloadArchive(ctx *context.APIContext) {
	var tp git.ArchiveType
	switch ballType := ctx.Params("ball_type"); ballType {
	case "tarball":
		tp = git.TARGZ
	case "zipball":
		tp = git.ZIP
	case "bundle":
		tp = git.BUNDLE
	default:
		ctx.Error(http.StatusBadRequest, "", fmt.Sprintf("Unknown archive type: %s", ballType))
		return
	}

	if ctx.Repo.GitRepo == nil {
		gitRepo, err := gitrepo.OpenRepository(ctx, ctx.Repo.Repository)
		if err != nil {
			ctx.Error(http.StatusInternalServerError, "OpenRepository", err)
			return
		}
		ctx.Repo.GitRepo = gitRepo
		defer gitRepo.Close()
	}

	r, err := archiver_service.NewRequest(ctx, ctx.Repo.Repository.ID, ctx.Repo.GitRepo, ctx.Params("*"), tp)
	if err != nil {
		ctx.ServerError("NewRequest", err)
		return
	}

	archive, err := r.Await(ctx)
	if err != nil {
		ctx.ServerError("archive.Await", err)
		return
	}

	download(ctx, r.GetArchiveName(), archive)
}
