// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"errors"
	"net/http"

	"forgejo.org/modules/base"
	"forgejo.org/services/context"
	contributors_service "forgejo.org/services/repository"
)

const (
	tplRecentCommits base.TplName = "repo/activity"
)

// RecentCommits renders the page to show recent commit frequency on repository
func RecentCommits(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("repo.activity.navbar.recent_commits")

	ctx.Data["PageIsActivity"] = true
	ctx.Data["PageIsRecentCommits"] = true
	ctx.PageData["repoLink"] = ctx.Repo.RepoLink

	ctx.HTML(http.StatusOK, tplRecentCommits)
}

// RecentCommitsData returns JSON of recent commits data
func RecentCommitsData(ctx *context.Context) {
	if contributorStats, err := contributors_service.GetContributorStats(ctx, ctx.Cache, ctx.Repo.Repository, ctx.Repo.CommitID); err != nil {
		if errors.Is(err, contributors_service.ErrAwaitGeneration) {
			ctx.Status(http.StatusAccepted)
			return
		}
		ctx.ServerError("RecentCommitsData", err)
	} else {
		ctx.JSON(http.StatusOK, contributorStats["total"].Weeks)
	}
}
