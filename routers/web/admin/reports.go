// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package admin

import (
	"fmt"
	"net/http"

	"forgejo.org/models/issues"
	"forgejo.org/models/moderation"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/user"
	"forgejo.org/modules/base"
	"forgejo.org/services/context"
	moderation_service "forgejo.org/services/moderation"
)

const (
	tplModerationReports       base.TplName = "admin/moderation/reports"
	tplModerationReportDetails base.TplName = "admin/moderation/report_details"
)

// AbuseReports renders the reports overview page from admin moderation section.
func AbuseReports(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("admin.moderation.reports")
	ctx.Data["PageIsAdminModerationReports"] = true

	reports, err := moderation.GetOpenReports(ctx)
	if err != nil {
		ctx.ServerError("Failed to load abuse reports", err)
		return
	}

	ctx.Data["Reports"] = reports
	ctx.Data["AbuseCategories"] = moderation.AbuseCategoriesTranslationKeys
	ctx.Data["GhostUserName"] = user.GhostUserName

	ctx.HTML(http.StatusOK, tplModerationReports)
}

// AbuseReportDetails renders a report details page opened from the reports overview from admin moderation section.
func AbuseReportDetails(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("admin.moderation.reports")
	ctx.Data["PageIsAdminModerationReports"] = true

	ctx.Data["Type"] = ctx.ParamsInt64(":type")
	ctx.Data["ID"] = ctx.ParamsInt64(":id")

	contentType := moderation.ReportedContentType(ctx.ParamsInt64(":type"))

	if !contentType.IsValid() {
		ctx.Flash.Error("Invalid content type")
		return
	}

	reports, err := moderation.GetOpenReportsByTypeAndContentID(ctx, contentType, ctx.ParamsInt64(":id"))
	if err != nil {
		ctx.ServerError("Failed to load reports", err)
		return
	}
	if len(reports) == 0 {
		// something is wrong
		ctx.HTML(http.StatusOK, tplModerationReportDetails)
		return
	}

	ctx.Data["Reports"] = reports
	ctx.Data["AbuseCategories"] = moderation.AbuseCategoriesTranslationKeys
	ctx.Data["GhostUserName"] = user.GhostUserName

	ctx.Data["GetShadowCopyMap"] = moderation_service.GetShadowCopyMap

	if err = setReportedContentDetails(ctx, reports[0]); err != nil {
		if user.IsErrUserNotExist(err) || issues.IsErrCommentNotExist(err) || issues.IsErrIssueNotExist(err) || repo_model.IsErrRepoNotExist(err) {
			ctx.Data["ContentReference"] = "Reported content no longer exists"
		} else {
			ctx.ServerError("Failed to load reported content details", err)
			return
		}
	}

	ctx.HTML(http.StatusOK, tplModerationReportDetails)
}

// setReportedContentDetails adds some values into context data for the given report
// (icon name, a reference, the URL and in case of issues and comments also the poster name).
func setReportedContentDetails(ctx *context.Context, report *moderation.AbuseReportDetailed) error {
	contentReference := ""
	var contentURL string
	var poster string
	contentType := report.ContentType
	contentID := report.ContentID

	ctx.Data["ContentTypeIconName"] = report.ContentTypeIconName()

	switch contentType {
	case moderation.ReportedContentTypeUser:
		reportedUser, err := user.GetUserByID(ctx, contentID)
		if err != nil {
			return err
		}

		contentReference = reportedUser.Name
		contentURL = reportedUser.HomeLink()
	case moderation.ReportedContentTypeRepository:
		repo, err := repo_model.GetRepositoryByID(ctx, contentID)
		if err != nil {
			return err
		}

		contentReference = repo.FullName()
		contentURL = repo.Link()
	case moderation.ReportedContentTypeIssue:
		issue, err := issues.GetIssueByID(ctx, contentID)
		if err != nil {
			return err
		}
		if err = issue.LoadRepo(ctx); err != nil {
			return err
		}
		if err = issue.LoadPoster(ctx); err != nil {
			return err
		}
		if issue.Poster != nil {
			poster = issue.Poster.Name
		}

		contentReference = fmt.Sprintf("%s#%d", issue.Repo.FullName(), issue.Index)
		contentURL = issue.Link()
	case moderation.ReportedContentTypeComment:
		comment, err := issues.GetCommentByID(ctx, contentID)
		if err != nil {
			return err
		}
		if err = comment.LoadIssue(ctx); err != nil {
			return err
		}
		if err = comment.Issue.LoadRepo(ctx); err != nil {
			return err
		}
		if err = comment.LoadPoster(ctx); err != nil {
			return err
		}
		if comment.Poster != nil {
			poster = comment.Poster.Name
		}

		contentURL = comment.Link(ctx)
		contentReference = contentURL
	}

	ctx.Data["ContentReference"] = contentReference
	ctx.Data["ContentURL"] = contentURL
	ctx.Data["Poster"] = poster
	return nil
}
