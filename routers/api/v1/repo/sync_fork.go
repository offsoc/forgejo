// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"net/http"

	git_model "forgejo.org/models/git"
	"forgejo.org/services/context"
	repo_service "forgejo.org/services/repository"
)

func getSyncForkInfo(ctx *context.APIContext, branch string) {
	if !ctx.Repo.Repository.IsFork {
		ctx.Error(http.StatusBadRequest, "NoFork", "The Repo must be a fork")
		return
	}

	syncForkInfo, err := repo_service.GetSyncForkInfo(ctx, ctx.Repo.Repository, branch)
	if err != nil {
		if git_model.IsErrBranchNotExist(err) {
			ctx.NotFound(err, branch)
			return
		}

		ctx.Error(http.StatusInternalServerError, "GetSyncForkInfo", err)
		return
	}

	ctx.JSON(http.StatusOK, syncForkInfo)
}

// SyncForkBranchInfo returns information about syncing the default fork branch with the base branch
func SyncForkDefaultInfo(ctx *context.APIContext) {
	// swagger:operation GET /repos/{owner}/{repo}/sync_fork repository repoSyncForkDefaultInfo
	// ---
	// summary: Gets information about syncing the fork default branch with the base branch
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/SyncForkInfo"
	//   "400":
	//     "$ref": "#/responses/error"
	//   "404":
	//     "$ref": "#/responses/notFound"
	getSyncForkInfo(ctx, ctx.Repo.Repository.DefaultBranch)
}

// SyncForkBranchInfo returns information about syncing a fork branch with the base branch
func SyncForkBranchInfo(ctx *context.APIContext) {
	// swagger:operation GET /repos/{owner}/{repo}/sync_fork/{branch} repository repoSyncForkBranchInfo
	// ---
	// summary: Gets information about syncing a fork branch with the base branch
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: branch
	//   in: path
	//   description: The branch
	//   type: string
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/SyncForkInfo"
	//   "400":
	//     "$ref": "#/responses/error"
	//   "404":
	//     "$ref": "#/responses/notFound"
	getSyncForkInfo(ctx, ctx.Params("branch"))
}

func syncForkBranch(ctx *context.APIContext, branch string) {
	if !ctx.Repo.Repository.IsFork {
		ctx.Error(http.StatusBadRequest, "NoFork", "The Repo must be a fork")
		return
	}

	syncForkInfo, err := repo_service.GetSyncForkInfo(ctx, ctx.Repo.Repository, branch)
	if err != nil {
		if git_model.IsErrBranchNotExist(err) {
			ctx.NotFound(err, branch)
			return
		}

		ctx.Error(http.StatusInternalServerError, "GetSyncForkInfo", err)
		return
	}

	if !syncForkInfo.Allowed {
		ctx.Error(http.StatusBadRequest, "NotAllowed", "You can't sync this branch")
		return
	}

	err = repo_service.SyncFork(ctx, ctx.Doer, ctx.Repo.Repository, branch)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "SyncFork", err)
		return
	}

	ctx.Status(http.StatusNoContent)
}

// SyncForkBranch syncs the default of a fork with the base branch
func SyncForkDefault(ctx *context.APIContext) {
	// swagger:operation POST /repos/{owner}/{repo}/sync_fork repository repoSyncForkDefault
	// ---
	// summary: Syncs the default branch of a fork with the base branch
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// responses:
	//   "204":
	//     "$ref": "#/responses/empty"
	//   "400":
	//     "$ref": "#/responses/error"
	//   "404":
	//     "$ref": "#/responses/notFound"
	syncForkBranch(ctx, ctx.Repo.Repository.DefaultBranch)
}

// SyncForkBranch syncs a fork branch with the base branch
func SyncForkBranch(ctx *context.APIContext) {
	// swagger:operation POST /repos/{owner}/{repo}/sync_fork/{branch} repository repoSyncForkBranch
	// ---
	// summary: Syncs a fork branch with the base branch
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: branch
	//   in: path
	//   description: The branch
	//   type: string
	//   required: true
	// responses:
	//   "204":
	//     "$ref": "#/responses/empty"
	//   "400":
	//     "$ref": "#/responses/error"
	//   "404":
	//     "$ref": "#/responses/notFound"
	syncForkBranch(ctx, ctx.Params("branch"))
}
