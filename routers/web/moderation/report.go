// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package moderation

import (
	"net/http"

	"code.gitea.io/gitea/models/moderation"
	"code.gitea.io/gitea/modules/base"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/services/context"
	"code.gitea.io/gitea/services/forms"
)

const (
	tplSubmitAbuseReport base.TplName = "moderation/new_abuse_report"
)

// NewReport renders the page for new abuse reports.
func NewReport(ctx *context.Context) {
	contentID := ctx.FormInt64("id")
	if contentID <= 0 {
		ctx.RenderWithErr(ctx.Tr("moderation.report_abuse_form.invalid"), tplSubmitAbuseReport, nil)
		log.Warn("The content ID is expected to be an integer greater that 0; the provided value is %d.", contentID)
		return
	}

	contentTypeString := ctx.FormString("type")
	var contentType moderation.ReportedContentType
	switch contentTypeString {
	case "user", "org":
		contentType = moderation.ReportedContentTypeUser
	case "repo":
		contentType = moderation.ReportedContentTypeRepository
	case "issue", "pull":
		contentType = moderation.ReportedContentTypeIssue
	case "comment":
		contentType = moderation.ReportedContentTypeComment
	default:
		ctx.RenderWithErr(ctx.Tr("moderation.report_abuse_form.invalid"), tplSubmitAbuseReport, nil)
		log.Warn("The provided content type `%s` is not among the expected values.", contentTypeString)
		return
	}

	setContextDataAndRender(ctx, contentType, contentID)
}

// setContextDataAndRender adds some values into context data and renders the new abuse report page.
func setContextDataAndRender(ctx *context.Context, contentType moderation.ReportedContentType, contentID int64) {
	ctx.Data["Title"] = ctx.Tr("moderation.report_abuse")
	ctx.Data["ContentID"] = contentID
	ctx.Data["ContentType"] = contentType
	ctx.Data["AbuseCategories"] = moderation.GetAbuseCategoriesList()
	ctx.Data["CancelLink"] = ctx.Doer.DashboardLink()
	ctx.HTML(http.StatusOK, tplSubmitAbuseReport)
}

// CreatePost handles the POST for creating a new abuse report.
func CreatePost(ctx *context.Context) {
	form := *web.GetForm(ctx).(*forms.ReportAbuseForm)

	if form.ContentID <= 0 || form.ContentType == 0 {
		ctx.RenderWithErr(ctx.Tr("moderation.report_abuse_form.invalid"), tplSubmitAbuseReport, nil)
		return
	}

	if ctx.HasError() {
		setContextDataAndRender(ctx, form.ContentType, form.ContentID)
		return
	}

	report := moderation.AbuseReport{
		ReporterID:  ctx.Doer.ID,
		ContentType: form.ContentType,
		ContentID:   form.ContentID,
		Category:    form.AbuseCategory,
		Remarks:     form.Remarks,
	}

	if err := moderation.ReportAbuse(ctx, &report); err != nil {
		ctx.ServerError("Something went wrong while trying to submit the new abuse report.", err)
		return
	}

	ctx.Flash.Success(ctx.Tr("moderation.reported_thank_you"))
	ctx.Redirect(ctx.Doer.DashboardLink())
}
