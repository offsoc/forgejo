// Copyright 2022 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package feed

import (
	"time"

	activities_model "code.gitea.io/gitea/models/activities"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/modules/context"

	"github.com/gorilla/feeds"
)

// ShowRepoFeed shows user activity on the repo as RSS / Atom feed
func ShowRepoFeed(ctx *context.Context, repo *repo_model.Repository, formatType string) {
	actions, err := activities_model.GetFeeds(ctx, activities_model.GetFeedsOptions{
		RequestedRepo:  repo,
		Actor:          ctx.Doer,
		IncludePrivate: true,
		Date:           ctx.FormString("date"),
	})
	if err != nil {
		ctx.ServerError("GetFeeds", err)
		return
	}

	feed := &feeds.Feed{
		Title:       ctx.Tr("home.feed_of", repo.FullName()),
		Link:        &feeds.Link{Href: repo.HTMLURL()},
		Description: repo.Description,
		Created:     time.Now(),
	}

	feed.Items, err = feedActionsToFeedItems(ctx, actions)
	if err != nil {
		ctx.ServerError("convert feed", err)
		return
	}

	writeFeed(ctx, feed, formatType)
}
