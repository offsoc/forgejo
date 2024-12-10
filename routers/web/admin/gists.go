// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package admin

import (
	"net/http"

	gist_model "code.gitea.io/gitea/models/gist"
	"code.gitea.io/gitea/modules/base"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/services/context"
	gist_service "code.gitea.io/gitea/services/gist"
)

const (
	tplGistsList base.TplName = "admin/gist/list"
)

// Gists shows all gists
func Gists(ctx *context.Context) {
	opts := new(gist_model.SearchGistOptions)

	opts.Actor = ctx.Doer
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

	gists, count, err := gist_model.SearchGist(ctx, opts)
	if err != nil {
		ctx.ServerError("SearchGist", err)
		return
	}

	err = gists.LoadOwner(ctx)
	if err != nil {
		ctx.ServerError("LoadOwner", err)
		return
	}

	ctx.Data["Title"] = ctx.Tr("admin.gists.title")
	ctx.Data["PageIsAdminGists"] = true
	ctx.Data["Gists"] = gists
	ctx.Data["Total"] = count

	pager := context.NewPagination(int(count), setting.UI.PackagesPagingNum, opts.Page, 5)
	pager.AddParamString("q", opts.Keyword)
	pager.AddParamString("sort", opts.SortOrder)
	ctx.Data["Page"] = pager

	ctx.HTML(http.StatusOK, tplGistsList)
}

// DeleteGist deletes a gist
func DeleteGist(ctx *context.Context) {
	gist, err := gist_model.GetGistByUUID(ctx, ctx.FormString("id"))
	if err != nil {
		ctx.ServerError("GetGistByUUID", err)
		return
	}

	err = gist_service.DeleteGist(ctx, gist)
	if err != nil {
		ctx.ServerError("DeleteGist", err)
		return
	}

	ctx.Flash.Success(ctx.Tr("gist.delete.success"))

	ctx.JSONRedirect("/admin/gists")
}
