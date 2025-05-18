// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package admin

import (
	"net/http"

	"forgejo.org/models/moderation"
	"forgejo.org/models/user"
	"forgejo.org/modules/base"
	"forgejo.org/services/context"
)

const tplModerationReports base.TplName = "admin/moderation/reports"

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
