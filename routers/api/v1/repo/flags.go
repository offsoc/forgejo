// Copyright 2024 The Forgejo Authors c/o Codeberg e.V.. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"net/http"

	api "forgejo.org/modules/structs"
	"forgejo.org/modules/web"
	"forgejo.org/services/context"
)

func ListFlags(ctx *context.APIContext) {
	// swagger:operation GET /repos/{owner}/{repo}/flags repository repoListFlags
	// ---
	// summary: List a repository's flags
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
	//     "$ref": "#/responses/StringSlice"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"

	repoFlags, err := ctx.Repo.Repository.ListFlags(ctx)
	if err != nil {
		ctx.InternalServerError(err)
		return
	}

	flags := make([]string, len(repoFlags))
	for i := range repoFlags {
		flags[i] = repoFlags[i].Name
	}

	ctx.SetTotalCountHeader(int64(len(repoFlags)))
	ctx.JSON(http.StatusOK, flags)
}

func ReplaceAllFlags(ctx *context.APIContext) {
	// swagger:operation PUT /repos/{owner}/{repo}/flags repository repoReplaceAllFlags
	// ---
	// summary: Replace all flags of a repository
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
	// - name: body
	//   in: body
	//   schema:
	//     "$ref": "#/definitions/ReplaceFlagsOption"
	// responses:
	//   "204":
	//     "$ref": "#/responses/empty"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"

	flagsForm := web.GetForm(ctx).(*api.ReplaceFlagsOption)

	if err := ctx.Repo.Repository.ReplaceAllFlags(ctx, flagsForm.Flags); err != nil {
		ctx.InternalServerError(err)
		return
	}

	ctx.Status(http.StatusNoContent)
}

func DeleteAllFlags(ctx *context.APIContext) {
	// swagger:operation DELETE /repos/{owner}/{repo}/flags repository repoDeleteAllFlags
	// ---
	// summary: Remove all flags from a repository
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
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"

	if err := ctx.Repo.Repository.ReplaceAllFlags(ctx, nil); err != nil {
		ctx.InternalServerError(err)
		return
	}

	ctx.Status(http.StatusNoContent)
}

func HasFlag(ctx *context.APIContext) {
	// swagger:operation GET /repos/{owner}/{repo}/flags/{flag} repository repoCheckFlag
	// ---
	// summary: Check if a repository has a given flag
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
	// - name: flag
	//   in: path
	//   description: name of the flag
	//   type: string
	//   required: true
	// responses:
	//   "204":
	//     "$ref": "#/responses/empty"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"

	hasFlag := ctx.Repo.Repository.HasFlag(ctx, ctx.Params(":flag"))
	if hasFlag {
		ctx.Status(http.StatusNoContent)
	} else {
		ctx.NotFound()
	}
}

func AddFlag(ctx *context.APIContext) {
	// swagger:operation PUT /repos/{owner}/{repo}/flags/{flag} repository repoAddFlag
	// ---
	// summary: Add a flag to a repository
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
	// - name: flag
	//   in: path
	//   description: name of the flag
	//   type: string
	//   required: true
	// responses:
	//   "204":
	//     "$ref": "#/responses/empty"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"

	flag := ctx.Params(":flag")

	if ctx.Repo.Repository.HasFlag(ctx, flag) {
		ctx.Status(http.StatusNoContent)
		return
	}

	if err := ctx.Repo.Repository.AddFlag(ctx, flag); err != nil {
		ctx.InternalServerError(err)
		return
	}
	ctx.Status(http.StatusNoContent)
}

func DeleteFlag(ctx *context.APIContext) {
	// swagger:operation DELETE /repos/{owner}/{repo}/flags/{flag} repository repoDeleteFlag
	// ---
	// summary: Remove a flag from a repository
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
	// - name: flag
	//   in: path
	//   description: name of the flag
	//   type: string
	//   required: true
	// responses:
	//   "204":
	//     "$ref": "#/responses/empty"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"

	flag := ctx.Params(":flag")

	if _, err := ctx.Repo.Repository.DeleteFlag(ctx, flag); err != nil {
		ctx.InternalServerError(err)
		return
	}
	ctx.Status(http.StatusNoContent)
}
