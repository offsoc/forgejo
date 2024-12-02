// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package explore

import (
	"net/http"

	gist_model "code.gitea.io/gitea/models/gist"
	"code.gitea.io/gitea/modules/base"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/services/context"
)

const (
	tplExploreGists base.TplName = "explore/gists"
)

func Gists(ctx *context.Context) {
	opts := new(gist_model.SearchGistOptions)

	opts.PageSize = setting.UI.ExplorePagingNum

	opts.Page = ctx.FormInt("page")
	if opts.Page <= 0 {
		opts.Page = 1
	}

	sortOrder := ctx.FormString("sort")
	if sortOrder == "" {
		sortOrder = setting.UI.ExploreDefaultSort
	}
	ctx.Data["SortType"] = sortOrder
	opts.SortOrder = sortOrder

	opts.Keyword = ctx.FormTrim("q")

	gists, count, err := gist_model.SearchGist(ctx, ctx.Doer, opts)
	if err != nil {
		ctx.ServerError("SearchGist", err)
		return
	}

	err = gists.LoadOwner(ctx)
	if err != nil {
		ctx.ServerError("LoadOwner", err)
		return
	}

	pager := context.NewPagination(int(count), opts.PageSize, opts.Page, 5)
	pager.SetDefaultParams(ctx)

	ctx.Data["PageIsExploreGists"] = true
	ctx.Data["Keyword"] = opts.Keyword
	ctx.Data["Gists"] = gists
	ctx.Data["Page"] = pager

	ctx.HTML(http.StatusOK, tplExploreGists)
}
