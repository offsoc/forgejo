// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package private

import (
	"fmt"
	"net/http"

	repo_model "forgejo.org/models/repo"
	"forgejo.org/modules/gitrepo"
	"forgejo.org/modules/private"
	gitea_context "forgejo.org/services/context"
)

// SetDefaultBranch updates the default branch
func SetDefaultBranch(ctx *gitea_context.PrivateContext) {
	ownerName := ctx.Params(":owner")
	repoName := ctx.Params(":repo")
	branch := ctx.Params(":branch")

	ctx.Repo.Repository.DefaultBranch = branch
	if err := gitrepo.SetDefaultBranch(ctx, ctx.Repo.Repository, ctx.Repo.Repository.DefaultBranch); err != nil {
		ctx.JSON(http.StatusInternalServerError, private.Response{
			Err: fmt.Sprintf("Unable to set default branch on repository: %s/%s Error: %v", ownerName, repoName, err),
		})
		return
	}

	if err := repo_model.UpdateDefaultBranch(ctx, ctx.Repo.Repository); err != nil {
		ctx.JSON(http.StatusInternalServerError, private.Response{
			Err: fmt.Sprintf("Unable to set default branch on repository: %s/%s Error: %v", ownerName, repoName, err),
		})
		return
	}
	ctx.PlainText(http.StatusOK, "success")
}
