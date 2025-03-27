// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package feed

import (
	"forgejo.org/services/context"
)

// RenderBranchFeed render format for branch or file
func RenderBranchFeed(feedType string) func(ctx *context.Context) {
	return func(ctx *context.Context) {
		if ctx.Repo.TreePath == "" {
			ShowBranchFeed(ctx, ctx.Repo.Repository, feedType)
		} else {
			ShowFileFeed(ctx, ctx.Repo.Repository, feedType)
		}
	}
}
