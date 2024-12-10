// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package gist

import (
	"net/http"

	gist_model "code.gitea.io/gitea/models/gist"
	api "code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/modules/util"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/routers/api/v1/utils"
	"code.gitea.io/gitea/services/context"
	"code.gitea.io/gitea/services/convert"
	gist_service "code.gitea.io/gitea/services/gist"
)

// Search for Gists
func Search(ctx *context.APIContext) {
	// swagger:operation GET /gists/search gist searchGists
	// ---
	// summary: Search for gists
	// produces:
	// - application/json
	// parameters:
	// - name: q
	//   in: query
	//   description: keyword
	//   type: string
	// - name: owner_id
	//   in: query
	//   description: search only for repos that the user with the given id owns
	//   type: integer
	//   format: int64
	// - name: sort
	//   in: query
	//   description: sort gists by attribute
	//   type: string
	//   enum: [newest, oldest, alphabetically, reversealphabetically]
	// - name: page
	//   in: query
	//   description: page number of results to return (1-based)
	//   type: integer
	// - name: limit
	//   in: query
	//   description: page size of results
	//   type: integer
	// responses:
	//   "200":
	//     "$ref": "#/responses/GistList"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	opts := &gist_model.SearchGistOptions{
		ListOptions: utils.GetListOptions(ctx),
		Actor:       ctx.Doer,
		Keyword:     ctx.FormTrim("q"),
		OwnerID:     ctx.FormInt64("uid"),
		SortOrder:   ctx.FormTrim("sort"),
	}

	gists, count, err := gist_model.SearchGist(ctx, opts)
	if err != nil {
		ctx.ServerError("SearchGist", err)
		return
	}

	err = gists.LoadOwner(ctx)
	if err != nil {
		ctx.ServerError("LoadOwner", err)
		return
	}

	ctx.SetLinkHeader(int(count), opts.PageSize)
	ctx.SetTotalCountHeader(count)
	ctx.JSON(http.StatusOK, convert.ToGistList(ctx, gists, ctx.Doer))
}

// Create a gist
func Create(ctx *context.APIContext) {
	// swagger:operation POST /gists gist createGist
	// ---
	// summary: Create a Gist
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: body
	//   in: body
	//   schema:
	//     "$ref": "#/definitions/CreateGistOption"
	// responses:
	//   "201":
	//     "$ref": "#/responses/Gist"
	//   "400":
	//     "$ref": "#/responses/error"
	//   "401":
	//     "$ref": "#/responses/unauthorized"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "422":
	//     "$ref": "#/responses/validationError"
	opt := web.GetForm(ctx).(*api.CreateGistOption)

	if len(opt.Files) == 0 {
		ctx.Error(http.StatusBadRequest, "files can't be empty", nil)
		return
	}

	visibility, err := gist_model.GistVisibilityFromName(opt.Visibility)
	if err != nil {
		ctx.Error(http.StatusBadRequest, "invalid visibility", nil)
		return
	}

	if len(opt.Files) == 0 {
		ctx.Error(http.StatusBadRequest, "no files", nil)
		return
	}

	files := make(map[string]string)
	for _, currentFile := range opt.Files {
		if util.PathContainsDirectory(currentFile.Name) {
			ctx.Error(http.StatusBadRequest, "invalid filename", nil)
			return
		}

		files[currentFile.Name] = currentFile.Content
	}

	gist, err := gist_service.CreateGist(ctx, ctx.Doer, opt.Name, opt.Description, visibility, files)
	if err != nil {
		ctx.ServerError("CreateGist", err)
		return
	}

	ctx.JSON(http.StatusCreated, convert.ToGist(ctx, gist, ctx.Doer))
}

// Get a gist
func Get(ctx *context.APIContext) {
	// swagger:operation GET /gists/{gistuuid} gist getGist
	// ---
	// summary: Get a Gist
	// produces:
	// - application/json
	// parameters:
	// - name: gistuuid
	//   in: path
	//   description: uuid of the gist
	//   type: string
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/Gist"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"
	err := ctx.Gist.LoadOwner(ctx)
	if err != nil {
		ctx.ServerError("LoadOwner", err)
		return
	}

	ctx.JSON(http.StatusOK, convert.ToGist(ctx, ctx.Gist, ctx.Doer))
}

// Get files of a gist
func GetFiles(ctx *context.APIContext) {
	// swagger:operation GET /gists/{gistuuid}/files gist getGistFiles
	// ---
	// summary: Get files of a Gist
	// produces:
	// - application/json
	// parameters:
	// - name: gistuuid
	//   in: path
	//   description: uuid of the gist
	//   type: string
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/GistFiles"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"
	files, err := gist_service.GetFiles(ctx, ctx.Gist)
	if err != nil {
		ctx.ServerError("GetFiles", err)
		return
	}

	ctx.JSON(http.StatusOK, files)
}

// Update files of a Gist
func UpdateFiles(ctx *context.APIContext) {
	// swagger:operation POST /gists/{gistuuid}/files gist updateGistFiles
	// ---
	// summary: Update files of a Gist
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: gistuuid
	//   in: path
	//   description: uuid of the gist
	//   type: string
	//   required: true
	// - name: body
	//   in: body
	//   schema:
	//     "$ref": "#/definitions/UpdateGistFilesOption"
	// responses:
	//   "204":
	//     "$ref": "#/responses/empty"
	//   "400":
	//     "$ref": "#/responses/error"
	//   "401":
	//     "$ref": "#/responses/unauthorized"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"
	//   "422":
	//     "$ref": "#/responses/validationError"
	opt := web.GetForm(ctx).(*api.UpdateGistFilesOption)

	for _, currentFile := range opt.Files {
		if util.PathContainsDirectory(currentFile.Name) {
			ctx.Error(http.StatusBadRequest, "invalid filename", nil)
			return
		}
	}

	err := gist_service.UpdateFiles(ctx, ctx.Gist, ctx.Doer, opt.Files)
	if err != nil {
		ctx.ServerError("UpdateFiles", err)
		return
	}

	ctx.Status(http.StatusNoContent)
}

// Delets a Gist
func Delete(ctx *context.APIContext) {
	// swagger:operation DELETE /gists/{gistuuid} gist deletGist
	// ---
	// summary: Deletes a Gist
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: gistuuid
	//   in: path
	//   description: uuid of the gist
	//   type: string
	//   required: true
	// responses:
	//   "204":
	//     "$ref": "#/responses/empty"
	//   "401":
	//     "$ref": "#/responses/unauthorized"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"
	err := gist_service.DeleteGist(ctx, ctx.Gist)
	if err != nil {
		ctx.ServerError("DeleteGist", err)
		return
	}

	ctx.Status(http.StatusNoContent)
}
