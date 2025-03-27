// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"net/http"
	"time"

	issues_model "forgejo.org/models/issues"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
	api "forgejo.org/modules/structs"
	"forgejo.org/modules/web"
	"forgejo.org/services/attachment"
	"forgejo.org/services/context"
	"forgejo.org/services/context/upload"
	"forgejo.org/services/convert"
	issue_service "forgejo.org/services/issue"
)

// GetIssueCommentAttachment gets a single attachment of the comment
func GetIssueCommentAttachment(ctx *context.APIContext) {
	// swagger:operation GET /repos/{owner}/{repo}/issues/comments/{id}/assets/{attachment_id} issue issueGetIssueCommentAttachment
	// ---
	// summary: Get a comment attachment
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
	//   description: id of the comment
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
	//     "$ref": "#/responses/error"

	comment := ctx.Comment
	attachment := getIssueCommentAttachmentSafeRead(ctx)
	if attachment == nil {
		return
	}
	if attachment.CommentID != comment.ID {
		log.Debug("User requested attachment[%d] is not in comment[%d].", attachment.ID, comment.ID)
		ctx.NotFound("attachment not in comment")
		return
	}

	ctx.JSON(http.StatusOK, convert.ToAPIAttachment(ctx.Repo.Repository, attachment))
}

// ListIssueCommentAttachments lists all attachments of the comment
func ListIssueCommentAttachments(ctx *context.APIContext) {
	// swagger:operation GET /repos/{owner}/{repo}/issues/comments/{id}/assets issue issueListIssueCommentAttachments
	// ---
	// summary: List comment's attachments
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
	//   description: id of the comment
	//   type: integer
	//   format: int64
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/AttachmentList"
	//   "404":
	//     "$ref": "#/responses/error"
	comment := ctx.Comment

	if err := comment.LoadAttachments(ctx); err != nil {
		ctx.Error(http.StatusInternalServerError, "LoadAttachments", err)
		return
	}

	ctx.JSON(http.StatusOK, convert.ToAPIAttachments(ctx.Repo.Repository, comment.Attachments))
}

// CreateIssueCommentAttachment creates an attachment and saves the given file
func CreateIssueCommentAttachment(ctx *context.APIContext) {
	// swagger:operation POST /repos/{owner}/{repo}/issues/comments/{id}/assets issue issueCreateIssueCommentAttachment
	// ---
	// summary: Create a comment attachment
	// produces:
	// - application/json
	// consumes:
	// - multipart/form-data
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
	//   description: id of the comment
	//   type: integer
	//   format: int64
	//   required: true
	// - name: name
	//   in: query
	//   description: name of the attachment
	//   type: string
	//   required: false
	// - name: updated_at
	//   in: query
	//   description: time of the attachment's creation. This is a timestamp in RFC 3339 format
	//   type: string
	//   format: date-time
	// - name: attachment
	//   in: formData
	//   description: attachment to upload
	//   type: file
	//   required: true
	// responses:
	//   "201":
	//     "$ref": "#/responses/Attachment"
	//   "400":
	//     "$ref": "#/responses/error"
	//   "404":
	//     "$ref": "#/responses/error"
	//   "413":
	//     "$ref": "#/responses/quotaExceeded"
	//   "422":
	//     "$ref": "#/responses/validationError"
	//   "423":
	//     "$ref": "#/responses/repoArchivedError"

	// Check if comment exists and load comment

	if !canUserWriteIssueCommentAttachment(ctx) {
		return
	}

	comment := ctx.Comment

	updatedAt := ctx.Req.FormValue("updated_at")
	if len(updatedAt) != 0 {
		updated, err := time.Parse(time.RFC3339, updatedAt)
		if err != nil {
			ctx.Error(http.StatusInternalServerError, "time.Parse", err)
			return
		}
		err = comment.LoadIssue(ctx)
		if err != nil {
			ctx.Error(http.StatusInternalServerError, "LoadIssue", err)
			return
		}
		err = issue_service.SetIssueUpdateDate(ctx, comment.Issue, &updated, ctx.Doer)
		if err != nil {
			ctx.Error(http.StatusForbidden, "SetIssueUpdateDate", err)
			return
		}
	}

	// Get uploaded file from request
	file, header, err := ctx.Req.FormFile("attachment")
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "FormFile", err)
		return
	}
	defer file.Close()

	filename := header.Filename
	if query := ctx.FormString("name"); query != "" {
		filename = query
	}

	attachment, err := attachment.UploadAttachment(ctx, file, setting.Attachment.AllowedTypes, header.Size, &repo_model.Attachment{
		Name:        filename,
		UploaderID:  ctx.Doer.ID,
		RepoID:      ctx.Repo.Repository.ID,
		IssueID:     comment.IssueID,
		CommentID:   comment.ID,
		NoAutoTime:  comment.Issue.NoAutoTime,
		CreatedUnix: comment.Issue.UpdatedUnix,
	})
	if err != nil {
		if upload.IsErrFileTypeForbidden(err) {
			ctx.Error(http.StatusUnprocessableEntity, "", err)
		} else {
			ctx.Error(http.StatusInternalServerError, "UploadAttachment", err)
		}
		return
	}

	if err := comment.LoadAttachments(ctx); err != nil {
		ctx.Error(http.StatusInternalServerError, "LoadAttachments", err)
		return
	}

	if err = issue_service.UpdateComment(ctx, comment, comment.ContentVersion, ctx.Doer, comment.Content); err != nil {
		ctx.ServerError("UpdateComment", err)
		return
	}

	ctx.JSON(http.StatusCreated, convert.ToAPIAttachment(ctx.Repo.Repository, attachment))
}

// EditIssueCommentAttachment updates the given attachment
func EditIssueCommentAttachment(ctx *context.APIContext) {
	// swagger:operation PATCH /repos/{owner}/{repo}/issues/comments/{id}/assets/{attachment_id} issue issueEditIssueCommentAttachment
	// ---
	// summary: Edit a comment attachment
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
	//   description: id of the comment
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
	//     "$ref": "#/responses/error"
	//   "413":
	//     "$ref": "#/responses/quotaExceeded"
	//   "423":
	//     "$ref": "#/responses/repoArchivedError"
	attach := getIssueCommentAttachmentSafeWrite(ctx)
	if attach == nil {
		return
	}

	form := web.GetForm(ctx).(*api.EditAttachmentOptions)
	if form.Name != "" {
		attach.Name = form.Name
	}

	if err := repo_model.UpdateAttachment(ctx, attach); err != nil {
		ctx.Error(http.StatusInternalServerError, "UpdateAttachment", attach)
	}
	ctx.JSON(http.StatusCreated, convert.ToAPIAttachment(ctx.Repo.Repository, attach))
}

// DeleteIssueCommentAttachment delete a given attachment
func DeleteIssueCommentAttachment(ctx *context.APIContext) {
	// swagger:operation DELETE /repos/{owner}/{repo}/issues/comments/{id}/assets/{attachment_id} issue issueDeleteIssueCommentAttachment
	// ---
	// summary: Delete a comment attachment
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
	//   description: id of the comment
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
	//     "$ref": "#/responses/error"
	//   "423":
	//     "$ref": "#/responses/repoArchivedError"
	attach := getIssueCommentAttachmentSafeWrite(ctx)
	if attach == nil {
		return
	}

	if err := repo_model.DeleteAttachment(ctx, attach, true); err != nil {
		ctx.Error(http.StatusInternalServerError, "DeleteAttachment", err)
		return
	}
	ctx.Status(http.StatusNoContent)
}

func getIssueCommentAttachmentSafeWrite(ctx *context.APIContext) *repo_model.Attachment {
	if !canUserWriteIssueCommentAttachment(ctx) {
		return nil
	}
	return getIssueCommentAttachmentSafeRead(ctx)
}

func canUserWriteIssueCommentAttachment(ctx *context.APIContext) bool {
	// ctx.Comment is assumed to be set in a safe way via a middleware
	comment := ctx.Comment

	canEditComment := ctx.IsSigned && (ctx.Doer.ID == comment.PosterID || ctx.IsUserRepoAdmin() || ctx.IsUserSiteAdmin()) && ctx.Repo.CanWriteIssuesOrPulls(comment.Issue.IsPull)
	if !canEditComment {
		ctx.Error(http.StatusForbidden, "", "user should have permission to edit comment")
		return false
	}

	return true
}

func getIssueCommentAttachmentSafeRead(ctx *context.APIContext) *repo_model.Attachment {
	// ctx.Comment is assumed to be set in a safe way via a middleware
	comment := ctx.Comment

	attachment, err := repo_model.GetAttachmentByID(ctx, ctx.ParamsInt64("attachment_id"))
	if err != nil {
		ctx.NotFoundOrServerError("GetAttachmentByID", repo_model.IsErrAttachmentNotExist, err)
		return nil
	}
	if !attachmentBelongsToRepoOrComment(ctx, attachment, comment) {
		return nil
	}
	return attachment
}

func attachmentBelongsToRepoOrComment(ctx *context.APIContext, attachment *repo_model.Attachment, comment *issues_model.Comment) bool {
	if attachment.RepoID != ctx.Repo.Repository.ID {
		log.Debug("Requested attachment[%d] does not belong to repo[%-v].", attachment.ID, ctx.Repo.Repository)
		ctx.NotFound("no such attachment in repo")
		return false
	}
	if attachment.IssueID == 0 || attachment.CommentID == 0 {
		log.Debug("Requested attachment[%d] is not in a comment.", attachment.ID)
		ctx.NotFound("no such attachment in comment")
		return false
	}
	if comment != nil && attachment.CommentID != comment.ID {
		log.Debug("Requested attachment[%d] does not belong to comment[%d].", attachment.ID, comment.ID)
		ctx.NotFound("no such attachment in comment")
		return false
	}
	return true
}
