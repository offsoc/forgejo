// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"errors"
	"fmt"
	"net/http"

	"forgejo.org/modules/git"
	api "forgejo.org/modules/structs"
	"forgejo.org/modules/web"
	"forgejo.org/services/context"
	"forgejo.org/services/convert"
)

// GetNote Get a note corresponding to a single commit from a repository
func GetNote(ctx *context.APIContext) {
	// swagger:operation GET /repos/{owner}/{repo}/git/notes/{sha} repository repoGetNote
	// ---
	// summary: Get a note corresponding to a single commit from a repository
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
	// - name: sha
	//   in: path
	//   description: a git ref or commit sha
	//   type: string
	//   required: true
	// - name: verification
	//   in: query
	//   description: include verification for every commit (disable for speedup, default 'true')
	//   type: boolean
	// - name: files
	//   in: query
	//   description: include a list of affected files for every commit (disable for speedup, default 'true')
	//   type: boolean
	// responses:
	//   "200":
	//     "$ref": "#/responses/Note"
	//   "422":
	//     "$ref": "#/responses/validationError"
	//   "404":
	//     "$ref": "#/responses/notFound"

	sha := ctx.Params(":sha")
	if !git.IsValidRefPattern(sha) {
		ctx.Error(http.StatusUnprocessableEntity, "no valid ref or sha", fmt.Sprintf("no valid ref or sha: %s", sha))
		return
	}
	getNote(ctx, sha)
}

func getNote(ctx *context.APIContext, identifier string) {
	if ctx.Repo.GitRepo == nil {
		ctx.InternalServerError(errors.New("no open git repo"))
		return
	}

	commitID, err := ctx.Repo.GitRepo.ConvertToGitID(identifier)
	if err != nil {
		if git.IsErrNotExist(err) {
			ctx.NotFound(err)
		} else {
			ctx.Error(http.StatusInternalServerError, "ConvertToSHA1", err)
		}
		return
	}

	var note git.Note
	if err := git.GetNote(ctx, ctx.Repo.GitRepo, commitID.String(), &note); err != nil {
		if git.IsErrNotExist(err) {
			ctx.NotFound(identifier)
			return
		}
		ctx.Error(http.StatusInternalServerError, "GetNote", err)
		return
	}

	verification := ctx.FormString("verification") == "" || ctx.FormBool("verification")
	files := ctx.FormString("files") == "" || ctx.FormBool("files")

	cmt, err := convert.ToCommit(ctx, ctx.Repo.Repository, ctx.Repo.GitRepo, note.Commit, nil,
		convert.ToCommitOptions{
			Stat:         true,
			Verification: verification,
			Files:        files,
		})
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "ToCommit", err)
		return
	}
	apiNote := api.Note{Message: string(note.Message), Commit: cmt}
	ctx.JSON(http.StatusOK, apiNote)
}

// SetNote Sets a note corresponding to a single commit from a repository
func SetNote(ctx *context.APIContext) {
	// swagger:operation POST /repos/{owner}/{repo}/git/notes/{sha} repository repoSetNote
	// ---
	// summary: Set a note corresponding to a single commit from a repository
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
	// - name: sha
	//   in: path
	//   description: a git ref or commit sha
	//   type: string
	//   required: true
	// - name: body
	//   in: body
	//   schema:
	//     "$ref": "#/definitions/NoteOptions"
	// responses:
	//   "200":
	//     "$ref": "#/responses/Note"
	//   "404":
	//     "$ref": "#/responses/notFound"
	//   "422":
	//     "$ref": "#/responses/validationError"
	sha := ctx.Params(":sha")
	if !git.IsValidRefPattern(sha) {
		ctx.Error(http.StatusUnprocessableEntity, "no valid ref or sha", fmt.Sprintf("no valid ref or sha: %s", sha))
		return
	}

	form := web.GetForm(ctx).(*api.NoteOptions)

	err := git.SetNote(ctx, ctx.Repo.GitRepo, sha, form.Message, ctx.Doer.Name, ctx.Doer.GetEmail())
	if err != nil {
		if git.IsErrNotExist(err) {
			ctx.NotFound(sha)
		} else {
			ctx.Error(http.StatusInternalServerError, "SetNote", err)
		}
		return
	}

	getNote(ctx, sha)
}

// RemoveNote Removes a note corresponding to a single commit from a repository
func RemoveNote(ctx *context.APIContext) {
	// swagger:operation DELETE /repos/{owner}/{repo}/git/notes/{sha} repository repoRemoveNote
	// ---
	// summary: Removes a note corresponding to a single commit from a repository
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
	// - name: sha
	//   in: path
	//   description: a git ref or commit sha
	//   type: string
	//   required: true
	// responses:
	//   "204":
	//     "$ref": "#/responses/empty"
	//   "404":
	//     "$ref": "#/responses/notFound"
	//   "422":
	//     "$ref": "#/responses/validationError"
	sha := ctx.Params(":sha")
	if !git.IsValidRefPattern(sha) {
		ctx.Error(http.StatusUnprocessableEntity, "no valid ref or sha", fmt.Sprintf("no valid ref or sha: %s", sha))
		return
	}

	err := git.RemoveNote(ctx, ctx.Repo.GitRepo, sha)
	if err != nil {
		if git.IsErrNotExist(err) {
			ctx.NotFound(sha)
		} else {
			ctx.Error(http.StatusInternalServerError, "RemoveNote", err)
		}
		return
	}

	ctx.Status(http.StatusNoContent)
}
