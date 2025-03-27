// Copyright 2018 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path"
	"strings"

	repo_model "forgejo.org/models/repo"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
	api "forgejo.org/modules/structs"
	"forgejo.org/modules/web"
	"forgejo.org/services/attachment"
	"forgejo.org/services/context"
	"forgejo.org/services/context/upload"
	"forgejo.org/services/convert"
)

func checkReleaseMatchRepo(ctx *context.APIContext, releaseID int64) bool {
	release, err := repo_model.GetReleaseByID(ctx, releaseID)
	if err != nil {
		if repo_model.IsErrReleaseNotExist(err) {
			ctx.NotFound()
			return false
		}
		ctx.Error(http.StatusInternalServerError, "GetReleaseByID", err)
		return false
	}
	if release.RepoID != ctx.Repo.Repository.ID {
		ctx.NotFound()
		return false
	}
	return true
}

// GetReleaseAttachment gets a single attachment of the release
func GetReleaseAttachment(ctx *context.APIContext) {
	// swagger:operation GET /repos/{owner}/{repo}/releases/{id}/assets/{attachment_id} repository repoGetReleaseAttachment
	// ---
	// summary: Get a release attachment
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
	// - name: id
	//   in: path
	//   description: id of the release
	//   type: integer
	//   format: int64
	//   required: true
	// - name: attachment_id
	//   in: path
	//   description: id of the attachment to get
	//   type: integer
	//   format: int64
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/Attachment"
	//   "404":
	//     "$ref": "#/responses/notFound"

	releaseID := ctx.ParamsInt64(":id")
	if !checkReleaseMatchRepo(ctx, releaseID) {
		return
	}

	attachID := ctx.ParamsInt64(":attachment_id")
	attach, err := repo_model.GetAttachmentByID(ctx, attachID)
	if err != nil {
		if repo_model.IsErrAttachmentNotExist(err) {
			ctx.NotFound()
			return
		}
		ctx.Error(http.StatusInternalServerError, "GetAttachmentByID", err)
		return
	}
	if attach.ReleaseID != releaseID {
		log.Info("User requested attachment is not in release, release_id %v, attachment_id: %v", releaseID, attachID)
		ctx.NotFound()
		return
	}
	// FIXME Should prove the existence of the given repo, but results in unnecessary database requests
	ctx.JSON(http.StatusOK, convert.ToAPIAttachment(ctx.Repo.Repository, attach))
}

// ListReleaseAttachments lists all attachments of the release
func ListReleaseAttachments(ctx *context.APIContext) {
	// swagger:operation GET /repos/{owner}/{repo}/releases/{id}/assets repository repoListReleaseAttachments
	// ---
	// summary: List release's attachments
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
	// - name: id
	//   in: path
	//   description: id of the release
	//   type: integer
	//   format: int64
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/AttachmentList"
	//   "404":
	//     "$ref": "#/responses/notFound"

	releaseID := ctx.ParamsInt64(":id")
	release, err := repo_model.GetReleaseByID(ctx, releaseID)
	if err != nil {
		if repo_model.IsErrReleaseNotExist(err) {
			ctx.NotFound()
			return
		}
		ctx.Error(http.StatusInternalServerError, "GetReleaseByID", err)
		return
	}
	if release.RepoID != ctx.Repo.Repository.ID {
		ctx.NotFound()
		return
	}
	if err := release.LoadAttributes(ctx); err != nil {
		ctx.Error(http.StatusInternalServerError, "LoadAttributes", err)
		return
	}
	ctx.JSON(http.StatusOK, convert.ToAPIRelease(ctx, ctx.Repo.Repository, release).Attachments)
}

// CreateReleaseAttachment creates an attachment and saves the given file
func CreateReleaseAttachment(ctx *context.APIContext) {
	// swagger:operation POST /repos/{owner}/{repo}/releases/{id}/assets repository repoCreateReleaseAttachment
	// ---
	// summary: Create a release attachment
	// produces:
	// - application/json
	// consumes:
	// - multipart/form-data
	// - application/octet-stream
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
	// - name: id
	//   in: path
	//   description: id of the release
	//   type: integer
	//   format: int64
	//   required: true
	// - name: name
	//   in: query
	//   description: name of the attachment
	//   type: string
	//   required: false
	// # There is no good way to specify "either 'attachment' or 'external_url' is required" with OpenAPI
	// # https://github.com/OAI/OpenAPI-Specification/issues/256
	// - name: attachment
	//   in: formData
	//   description: attachment to upload (this parameter is incompatible with `external_url`)
	//   type: file
	//   required: false
	// - name: external_url
	//   in: formData
	//   description: url to external asset (this parameter is incompatible with `attachment`)
	//   type: string
	//   required: false
	// responses:
	//   "201":
	//     "$ref": "#/responses/Attachment"
	//   "400":
	//     "$ref": "#/responses/error"
	//   "404":
	//     "$ref": "#/responses/notFound"
	//   "413":
	//     "$ref": "#/responses/quotaExceeded"

	// Check if attachments are enabled
	if !setting.Attachment.Enabled {
		ctx.NotFound("Attachment is not enabled")
		return
	}

	// Check if release exists an load release
	releaseID := ctx.ParamsInt64(":id")
	if !checkReleaseMatchRepo(ctx, releaseID) {
		return
	}

	// Get uploaded file from request
	var isForm, hasAttachmentFile, hasExternalURL bool
	externalURL := ctx.FormString("external_url")
	hasExternalURL = externalURL != ""
	filename := ctx.FormString("name")
	isForm = strings.HasPrefix(strings.ToLower(ctx.Req.Header.Get("Content-Type")), "multipart/form-data")

	if isForm {
		_, _, err := ctx.Req.FormFile("attachment")
		hasAttachmentFile = err == nil
	} else {
		hasAttachmentFile = ctx.Req.Body != nil
	}

	if hasAttachmentFile && hasExternalURL {
		ctx.Error(http.StatusBadRequest, "DuplicateAttachment", "'attachment' and 'external_url' are mutually exclusive")
	} else if hasAttachmentFile {
		var content io.ReadCloser
		var size int64 = -1

		if isForm {
			var header *multipart.FileHeader
			content, header, _ = ctx.Req.FormFile("attachment")
			size = header.Size
			defer content.Close()
			if filename == "" {
				filename = header.Filename
			}
		} else {
			content = ctx.Req.Body
			defer content.Close()
		}

		if filename == "" {
			ctx.Error(http.StatusBadRequest, "MissingName", "Missing 'name' parameter")
			return
		}

		// Create a new attachment and save the file
		attach, err := attachment.UploadAttachment(ctx, content, setting.Repository.Release.AllowedTypes, size, &repo_model.Attachment{
			Name:       filename,
			UploaderID: ctx.Doer.ID,
			RepoID:     ctx.Repo.Repository.ID,
			ReleaseID:  releaseID,
		})
		if err != nil {
			if upload.IsErrFileTypeForbidden(err) {
				ctx.Error(http.StatusBadRequest, "DetectContentType", err)
				return
			}
			ctx.Error(http.StatusInternalServerError, "NewAttachment", err)
			return
		}

		ctx.JSON(http.StatusCreated, convert.ToAPIAttachment(ctx.Repo.Repository, attach))
	} else if hasExternalURL {
		url, err := url.Parse(externalURL)
		if err != nil {
			ctx.Error(http.StatusBadRequest, "InvalidExternalURL", err)
			return
		}

		if filename == "" {
			filename = path.Base(url.Path)

			if filename == "." {
				// Url path is empty
				filename = url.Host
			}
		}

		attach, err := attachment.NewExternalAttachment(ctx, &repo_model.Attachment{
			Name:        filename,
			UploaderID:  ctx.Doer.ID,
			RepoID:      ctx.Repo.Repository.ID,
			ReleaseID:   releaseID,
			ExternalURL: url.String(),
		})
		if err != nil {
			if repo_model.IsErrInvalidExternalURL(err) {
				ctx.Error(http.StatusBadRequest, "NewExternalAttachment", err)
			} else {
				ctx.Error(http.StatusInternalServerError, "NewExternalAttachment", err)
			}
			return
		}

		ctx.JSON(http.StatusCreated, convert.ToAPIAttachment(ctx.Repo.Repository, attach))
	} else {
		ctx.Error(http.StatusBadRequest, "MissingAttachment", "One of 'attachment' or 'external_url' is required")
	}
}

// EditReleaseAttachment updates the given attachment
func EditReleaseAttachment(ctx *context.APIContext) {
	// swagger:operation PATCH /repos/{owner}/{repo}/releases/{id}/assets/{attachment_id} repository repoEditReleaseAttachment
	// ---
	// summary: Edit a release attachment
	// produces:
	// - application/json
	// consumes:
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
	// - name: id
	//   in: path
	//   description: id of the release
	//   type: integer
	//   format: int64
	//   required: true
	// - name: attachment_id
	//   in: path
	//   description: id of the attachment to edit
	//   type: integer
	//   format: int64
	//   required: true
	// - name: body
	//   in: body
	//   schema:
	//     "$ref": "#/definitions/EditAttachmentOptions"
	// responses:
	//   "201":
	//     "$ref": "#/responses/Attachment"
	//   "404":
	//     "$ref": "#/responses/notFound"
	//   "413":
	//     "$ref": "#/responses/quotaExceeded"

	form := web.GetForm(ctx).(*api.EditAttachmentOptions)

	// Check if release exists an load release
	releaseID := ctx.ParamsInt64(":id")
	if !checkReleaseMatchRepo(ctx, releaseID) {
		return
	}

	attachID := ctx.ParamsInt64(":attachment_id")
	attach, err := repo_model.GetAttachmentByID(ctx, attachID)
	if err != nil {
		if repo_model.IsErrAttachmentNotExist(err) {
			ctx.NotFound()
			return
		}
		ctx.Error(http.StatusInternalServerError, "GetAttachmentByID", err)
		return
	}
	if attach.ReleaseID != releaseID {
		log.Info("User requested attachment is not in release, release_id %v, attachment_id: %v", releaseID, attachID)
		ctx.NotFound()
		return
	}
	// FIXME Should prove the existence of the given repo, but results in unnecessary database requests
	if form.Name != "" {
		attach.Name = form.Name
	}

	if form.DownloadURL != "" {
		if attach.ExternalURL == "" {
			ctx.Error(http.StatusBadRequest, "EditAttachment", "existing attachment is not external")
			return
		}
		attach.ExternalURL = form.DownloadURL
	}

	if err := repo_model.UpdateAttachment(ctx, attach); err != nil {
		if repo_model.IsErrInvalidExternalURL(err) {
			ctx.Error(http.StatusBadRequest, "UpdateAttachment", err)
		} else {
			ctx.Error(http.StatusInternalServerError, "UpdateAttachment", err)
		}
		return
	}
	ctx.JSON(http.StatusCreated, convert.ToAPIAttachment(ctx.Repo.Repository, attach))
}

// DeleteReleaseAttachment delete a given attachment
func DeleteReleaseAttachment(ctx *context.APIContext) {
	// swagger:operation DELETE /repos/{owner}/{repo}/releases/{id}/assets/{attachment_id} repository repoDeleteReleaseAttachment
	// ---
	// summary: Delete a release attachment
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
	// - name: id
	//   in: path
	//   description: id of the release
	//   type: integer
	//   format: int64
	//   required: true
	// - name: attachment_id
	//   in: path
	//   description: id of the attachment to delete
	//   type: integer
	//   format: int64
	//   required: true
	// responses:
	//   "204":
	//     "$ref": "#/responses/empty"
	//   "404":
	//     "$ref": "#/responses/notFound"

	// Check if release exists an load release
	releaseID := ctx.ParamsInt64(":id")
	if !checkReleaseMatchRepo(ctx, releaseID) {
		return
	}

	attachID := ctx.ParamsInt64(":attachment_id")
	attach, err := repo_model.GetAttachmentByID(ctx, attachID)
	if err != nil {
		if repo_model.IsErrAttachmentNotExist(err) {
			ctx.NotFound()
			return
		}
		ctx.Error(http.StatusInternalServerError, "GetAttachmentByID", err)
		return
	}
	if attach.ReleaseID != releaseID {
		log.Info("User requested attachment is not in release, release_id %v, attachment_id: %v", releaseID, attachID)
		ctx.NotFound()
		return
	}
	// FIXME Should prove the existence of the given repo, but results in unnecessary database requests

	if err := repo_model.DeleteAttachment(ctx, attach, true); err != nil {
		ctx.Error(http.StatusInternalServerError, "DeleteAttachment", err)
		return
	}
	ctx.Status(http.StatusNoContent)
}
