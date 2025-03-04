// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package shared

import (
	"html/template"
	"net/http"

	quota_model "code.gitea.io/gitea/models/quota"
	"code.gitea.io/gitea/modules/base"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/services/context"
)

// StorageOverview render a size overview of the user, as well as relevant
// quota limits of the instance.
func StorageOverview(ctx *context.Context, userID int64, tpl base.TplName) {
	if !setting.Quota.Enabled {
		ctx.NotFound("MustEnableQuota", nil)
	}
	ctx.Data["Title"] = ctx.Tr("settings.storage_overview")
	ctx.Data["PageIsStorageOverview"] = true

	ctx.Data["Color"] = func(subject quota_model.LimitSubject) float64 {
		return float64(subject) * 137.50776405003785 // Golden angle.
	}

	ctx.Data["PrettySubject"] = func(subject quota_model.LimitSubject) template.HTML {
		switch subject {
		case quota_model.LimitSubjectSizeAll:
			return ctx.Locale.Tr("settings.quota.sizes.all")
		case quota_model.LimitSubjectSizeReposAll:
			return ctx.Locale.Tr("settings.quota.sizes.repos.all")
		case quota_model.LimitSubjectSizeReposPublic:
			return ctx.Locale.Tr("settings.quota.sizes.repos.public")
		case quota_model.LimitSubjectSizeReposPrivate:
			return ctx.Locale.Tr("settings.quota.sizes.repos.private")
		case quota_model.LimitSubjectSizeGitAll:
			return ctx.Locale.Tr("settings.quota.sizes.git.all")
		case quota_model.LimitSubjectSizeGitLFS:
			return ctx.Locale.Tr("settings.quota.sizes.git.lfs")
		case quota_model.LimitSubjectSizeAssetsAll:
			return ctx.Locale.Tr("settings.quota.sizes.assets.all")
		case quota_model.LimitSubjectSizeAssetsAttachmentsAll:
			return ctx.Locale.Tr("settings.quota.sizes.assets.attachments.all")
		case quota_model.LimitSubjectSizeAssetsAttachmentsIssues:
			return ctx.Locale.Tr("settings.quota.sizes.assets.attachments.issues")
		case quota_model.LimitSubjectSizeAssetsAttachmentsReleases:
			return ctx.Locale.Tr("settings.quota.sizes.assets.attachments.releases")
		case quota_model.LimitSubjectSizeAssetsArtifacts:
			return ctx.Locale.Tr("settings.quota.sizes.assets.artifacts")
		case quota_model.LimitSubjectSizeAssetsPackagesAll:
			return ctx.Locale.Tr("settings.quota.sizes.assets.packages.all")
		case quota_model.LimitSubjectSizeWiki:
			return ctx.Locale.Tr("settings.quota.sizes.wiki")
		default:
			panic("unrecognized subject: " + subject.String())
		}
	}

	sizeUsed, err := quota_model.GetUsedForUser(ctx, userID)
	if err != nil {
		ctx.ServerError("GetUsedForUser", err)
		return
	}
	ctx.Data["SizeUsed"] = sizeUsed

	quotaGroups, err := quota_model.GetGroupsForUser(ctx, userID)
	if err != nil {
		ctx.ServerError("GetGroupsForUser", err)
		return
	}
	if len(quotaGroups) == 0 {
		quotaGroups = append(quotaGroups, &quota_model.Group{
			Name: "Global quota",
			Rules: []quota_model.Rule{
				{
					Name:     "Default",
					Limit:    setting.Quota.Default.Total,
					Subjects: quota_model.LimitSubjects{quota_model.LimitSubjectSizeAll},
				},
			},
		},
		)
	}
	ctx.Data["QuotaGroups"] = quotaGroups

	ctx.HTML(http.StatusOK, tpl)
}
