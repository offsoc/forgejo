// Copyright 2016 The Gogs Authors. All rights reserved.
// Copyright 2018 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"errors"
	"net/http"
	"reflect"

	issues_model "forgejo.org/models/issues"
	api "forgejo.org/modules/structs"
	"forgejo.org/modules/web"
	"forgejo.org/services/context"
	"forgejo.org/services/convert"
	issue_service "forgejo.org/services/issue"
)

// ListIssueLabels list all the labels of an issue
func ListIssueLabels(ctx *context.APIContext) {
	// swagger:operation GET /repos/{owner}/{repo}/issues/{index}/labels issue issueGetLabels
	// ---
	// summary: Get an issue's labels
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
	// - name: index
	//   in: path
	//   description: index of the issue
	//   type: integer
	//   format: int64
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/LabelList"
	//   "404":
	//     "$ref": "#/responses/notFound"

	issue, err := issues_model.GetIssueByIndex(ctx, ctx.Repo.Repository.ID, ctx.ParamsInt64(":index"))
	if err != nil {
		if issues_model.IsErrIssueNotExist(err) {
			ctx.NotFound()
		} else {
			ctx.Error(http.StatusInternalServerError, "GetIssueByIndex", err)
		}
		return
	}

	if err := issue.LoadAttributes(ctx); err != nil {
		ctx.Error(http.StatusInternalServerError, "LoadAttributes", err)
		return
	}

	ctx.JSON(http.StatusOK, convert.ToLabelList(issue.Labels, ctx.Repo.Repository, ctx.Repo.Owner))
}

// AddIssueLabels add labels for an issue
func AddIssueLabels(ctx *context.APIContext) {
	// swagger:operation POST /repos/{owner}/{repo}/issues/{index}/labels issue issueAddLabel
	// ---
	// summary: Add a label to an issue
	// consumes:
	// - application/json
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
	// - name: index
	//   in: path
	//   description: index of the issue
	//   type: integer
	//   format: int64
	//   required: true
	// - name: body
	//   in: body
	//   schema:
	//     "$ref": "#/definitions/IssueLabelsOption"
	// responses:
	//   "200":
	//     "$ref": "#/responses/LabelList"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"

	form := web.GetForm(ctx).(*api.IssueLabelsOption)
	issue, labels, err := prepareForReplaceOrAdd(ctx, *form)
	if err != nil {
		return
	}

	if err = issue_service.AddLabels(ctx, issue, ctx.Doer, labels); err != nil {
		ctx.Error(http.StatusInternalServerError, "AddLabels", err)
		return
	}

	labels, err = issues_model.GetLabelsByIssueID(ctx, issue.ID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "GetLabelsByIssueID", err)
		return
	}

	ctx.JSON(http.StatusOK, convert.ToLabelList(labels, ctx.Repo.Repository, ctx.Repo.Owner))
}

// DeleteIssueLabel delete a label for an issue
func DeleteIssueLabel(ctx *context.APIContext) {
	// swagger:operation DELETE /repos/{owner}/{repo}/issues/{index}/labels/{id} issue issueRemoveLabel
	// ---
	// summary: Remove a label from an issue
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
	// - name: index
	//   in: path
	//   description: index of the issue
	//   type: integer
	//   format: int64
	//   required: true
	// - name: id
	//   in: path
	//   description: id of the label to remove
	//   type: integer
	//   format: int64
	//   required: true
	// - name: body
	//   in: body
	//   schema:
	//     "$ref": "#/definitions/DeleteLabelsOption"
	// responses:
	//   "204":
	//     "$ref": "#/responses/empty"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"
	//   "422":
	//     "$ref": "#/responses/validationError"
	form := web.GetForm(ctx).(*api.DeleteLabelsOption)

	issue, err := issues_model.GetIssueByIndex(ctx, ctx.Repo.Repository.ID, ctx.ParamsInt64(":index"))
	if err != nil {
		if issues_model.IsErrIssueNotExist(err) {
			ctx.NotFound()
		} else {
			ctx.Error(http.StatusInternalServerError, "GetIssueByIndex", err)
		}
		return
	}

	if !ctx.Repo.CanWriteIssuesOrPulls(issue.IsPull) {
		ctx.Status(http.StatusForbidden)
		return
	}

	if err := issue_service.SetIssueUpdateDate(ctx, issue, form.Updated, ctx.Doer); err != nil {
		ctx.Error(http.StatusForbidden, "SetIssueUpdateDate", err)
		return
	}

	label, err := issues_model.GetLabelByID(ctx, ctx.ParamsInt64(":id"))
	if err != nil {
		if issues_model.IsErrLabelNotExist(err) {
			ctx.Error(http.StatusUnprocessableEntity, "", err)
		} else {
			ctx.Error(http.StatusInternalServerError, "GetLabelByID", err)
		}
		return
	}

	if err := issue_service.RemoveLabel(ctx, issue, ctx.Doer, label); err != nil {
		ctx.Error(http.StatusInternalServerError, "DeleteIssueLabel", err)
		return
	}

	ctx.Status(http.StatusNoContent)
}

// ReplaceIssueLabels replace labels for an issue
func ReplaceIssueLabels(ctx *context.APIContext) {
	// swagger:operation PUT /repos/{owner}/{repo}/issues/{index}/labels issue issueReplaceLabels
	// ---
	// summary: Replace an issue's labels
	// consumes:
	// - application/json
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
	// - name: index
	//   in: path
	//   description: index of the issue
	//   type: integer
	//   format: int64
	//   required: true
	// - name: body
	//   in: body
	//   schema:
	//     "$ref": "#/definitions/IssueLabelsOption"
	// responses:
	//   "200":
	//     "$ref": "#/responses/LabelList"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"
	form := web.GetForm(ctx).(*api.IssueLabelsOption)
	issue, labels, err := prepareForReplaceOrAdd(ctx, *form)
	if err != nil {
		return
	}

	if err := issue_service.ReplaceLabels(ctx, issue, ctx.Doer, labels); err != nil {
		ctx.Error(http.StatusInternalServerError, "ReplaceLabels", err)
		return
	}

	labels, err = issues_model.GetLabelsByIssueID(ctx, issue.ID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "GetLabelsByIssueID", err)
		return
	}

	ctx.JSON(http.StatusOK, convert.ToLabelList(labels, ctx.Repo.Repository, ctx.Repo.Owner))
}

// ClearIssueLabels delete all the labels for an issue
func ClearIssueLabels(ctx *context.APIContext) {
	// swagger:operation DELETE /repos/{owner}/{repo}/issues/{index}/labels issue issueClearLabels
	// ---
	// summary: Remove all labels from an issue
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
	// - name: index
	//   in: path
	//   description: index of the issue
	//   type: integer
	//   format: int64
	//   required: true
	// - name: body
	//   in: body
	//   schema:
	//     "$ref": "#/definitions/DeleteLabelsOption"
	// responses:
	//   "204":
	//     "$ref": "#/responses/empty"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"
	form := web.GetForm(ctx).(*api.DeleteLabelsOption)

	issue, err := issues_model.GetIssueByIndex(ctx, ctx.Repo.Repository.ID, ctx.ParamsInt64(":index"))
	if err != nil {
		if issues_model.IsErrIssueNotExist(err) {
			ctx.NotFound()
		} else {
			ctx.Error(http.StatusInternalServerError, "GetIssueByIndex", err)
		}
		return
	}

	if !ctx.Repo.CanWriteIssuesOrPulls(issue.IsPull) {
		ctx.Status(http.StatusForbidden)
		return
	}

	if err := issue_service.SetIssueUpdateDate(ctx, issue, form.Updated, ctx.Doer); err != nil {
		ctx.Error(http.StatusForbidden, "SetIssueUpdateDate", err)
		return
	}

	if err := issue_service.ClearLabels(ctx, issue, ctx.Doer); err != nil {
		ctx.Error(http.StatusInternalServerError, "ClearLabels", err)
		return
	}

	ctx.Status(http.StatusNoContent)
}

func prepareForReplaceOrAdd(ctx *context.APIContext, form api.IssueLabelsOption) (*issues_model.Issue, []*issues_model.Label, error) {
	issue, err := issues_model.GetIssueByIndex(ctx, ctx.Repo.Repository.ID, ctx.ParamsInt64(":index"))
	if err != nil {
		if issues_model.IsErrIssueNotExist(err) {
			ctx.NotFound()
		} else {
			ctx.Error(http.StatusInternalServerError, "GetIssueByIndex", err)
		}
		return nil, nil, err
	}

	var (
		labelIDs   []int64
		labelNames []string
	)
	for _, label := range form.Labels {
		rv := reflect.ValueOf(label)
		switch rv.Kind() {
		case reflect.Float64:
			labelIDs = append(labelIDs, int64(rv.Float()))
		case reflect.String:
			labelNames = append(labelNames, rv.String())
		default:
			ctx.Error(http.StatusBadRequest, "InvalidLabel", "a label must be an integer or a string")
			return nil, nil, errors.New("invalid label")
		}
	}
	if len(labelIDs) > 0 && len(labelNames) > 0 {
		ctx.Error(http.StatusBadRequest, "InvalidLabels", "labels should be an array of strings or integers")
		return nil, nil, errors.New("invalid labels")
	}
	if len(labelNames) > 0 {
		repoLabelIDs, err := issues_model.GetLabelIDsInRepoByNames(ctx, ctx.Repo.Repository.ID, labelNames)
		if err != nil {
			ctx.Error(http.StatusInternalServerError, "GetLabelIDsInRepoByNames", err)
			return nil, nil, err
		}
		labelIDs = append(labelIDs, repoLabelIDs...)
		if ctx.Repo.Owner.IsOrganization() {
			orgLabelIDs, err := issues_model.GetLabelIDsInOrgByNames(ctx, ctx.Repo.Owner.ID, labelNames)
			if err != nil {
				ctx.Error(http.StatusInternalServerError, "GetLabelIDsInOrgByNames", err)
				return nil, nil, err
			}
			labelIDs = append(labelIDs, orgLabelIDs...)
		}
	}

	labels, err := issues_model.GetLabelsByIDs(ctx, labelIDs, "id", "repo_id", "org_id", "name", "exclusive")
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "GetLabelsByIDs", err)
		return nil, nil, err
	}

	if !ctx.Repo.CanWriteIssuesOrPulls(issue.IsPull) {
		ctx.Status(http.StatusForbidden)
		return nil, nil, nil
	}

	err = issue_service.SetIssueUpdateDate(ctx, issue, form.Updated, ctx.Doer)
	if err != nil {
		ctx.Error(http.StatusForbidden, "SetIssueUpdateDate", err)
		return nil, nil, err
	}

	return issue, labels, err
}
