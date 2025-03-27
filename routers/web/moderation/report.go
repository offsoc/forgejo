// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package moderation

import (
	"errors"
	"net/http"

	"forgejo.org/models/moderation"
	"forgejo.org/modules/base"
	"forgejo.org/modules/log"
	"forgejo.org/modules/web"
	"forgejo.org/services/context"
	"forgejo.org/services/forms"
	moderation_service "forgejo.org/services/moderation"
)

const (
	tplSubmitAbuseReport base.TplName = "moderation/new_abuse_report"
)

// NewReport renders the page for new abuse reports.
func NewReport(ctx *context.Context) {
	contentID := ctx.FormInt64("id")
	if contentID <= 0 {
		setMinimalContextData(ctx)
		ctx.RenderWithErr(ctx.Tr("moderation.report_abuse_form.invalid"), tplSubmitAbuseReport, nil)
		log.Warn("The content ID is expected to be an integer greater that 0; the provided value is %s.", ctx.FormString("id"))
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
		setMinimalContextData(ctx)
		ctx.RenderWithErr(ctx.Tr("moderation.report_abuse_form.invalid"), tplSubmitAbuseReport, nil)
		log.Warn("The provided content type `%s` is not among the expected values.", contentTypeString)
		return
	}

	if moderation.AlreadyReportedByAndOpen(ctx, ctx.Doer.ID, contentType, contentID) {
		setMinimalContextData(ctx)
		ctx.RenderWithErr(ctx.Tr("moderation.report_abuse_form.already_reported"), tplSubmitAbuseReport, nil)
		return
	}

	setContextDataAndRender(ctx, contentType, contentID)
}

// setMinimalContextData adds minimal values (Title and CancelLink) into context data.
func setMinimalContextData(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("moderation.report_abuse")
	ctx.Data["CancelLink"] = ctx.Doer.DashboardLink()
}

// setContextDataAndRender adds some values into context data and renders the new abuse report page.
func setContextDataAndRender(ctx *context.Context, contentType moderation.ReportedContentType, contentID int64) {
	setMinimalContextData(ctx)
	ctx.Data["ContentID"] = contentID
	ctx.Data["ContentType"] = contentType
	ctx.Data["AbuseCategories"] = moderation.GetAbuseCategoriesList()
	ctx.HTML(http.StatusOK, tplSubmitAbuseReport)
}

// CreatePost handles the POST for creating a new abuse report.
func CreatePost(ctx *context.Context) {
	form := *web.GetForm(ctx).(*forms.ReportAbuseForm)

	if form.ContentID <= 0 || !form.ContentType.IsValid() {
		setMinimalContextData(ctx)
		ctx.RenderWithErr(ctx.Tr("moderation.report_abuse_form.invalid"), tplSubmitAbuseReport, nil)
		return
	}

	if ctx.HasError() {
		setContextDataAndRender(ctx, form.ContentType, form.ContentID)
		return
	}

	can, err := moderation_service.CanReport(*ctx, ctx.Doer, form.ContentType, form.ContentID)
	if err != nil {
		if errors.Is(err, moderation_service.ErrContentDoesNotExist) || errors.Is(err, moderation_service.ErrDoerNotAllowed) {
			ctx.Flash.Error(ctx.Tr("moderation.report_abuse_form.invalid"))
			ctx.Redirect(ctx.Doer.DashboardLink())
		} else {
			ctx.ServerError("Failed to check if user can report content", err)
		}
		return
	} else if !can {
		ctx.Flash.Error(ctx.Tr("moderation.report_abuse_form.invalid"))
		ctx.Redirect(ctx.Doer.DashboardLink())
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
		if errors.Is(err, moderation.ErrSelfReporting) {
			ctx.Flash.Error(ctx.Tr("moderation.reporting_failed", err))
			ctx.Redirect(ctx.Doer.DashboardLink())
		} else {
			ctx.ServerError("Failed to save new abuse report", err)
		}
		return
	}

	ctx.Flash.Success(ctx.Tr("moderation.reported_thank_you"))
	ctx.Redirect(ctx.Doer.DashboardLink())
}
