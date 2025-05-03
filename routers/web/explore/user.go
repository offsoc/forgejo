// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package explore

import (
	"bytes"
	"net/http"

	"forgejo.org/models/db"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/base"
	"forgejo.org/modules/container"
	"forgejo.org/modules/log"
	"forgejo.org/modules/optional"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/sitemap"
	"forgejo.org/modules/structs"
	"forgejo.org/services/context"
)

const (
	// `tplExploreUsers` explore users page template.
	tplExploreUsers base.TplName = "explore/users"
)

var nullByte = []byte{0x00}

func isKeywordValid(keyword string) bool {
	return !bytes.Contains([]byte(keyword), nullByte)
}

// `RenderUserSearch` render user search page.
func RenderUserSearch(ctx *context.Context, opts *user_model.SearchUserOptions, tplName base.TplName) {
	// Sitemap index for sitemap paths.
	opts.Page = int(ctx.ParamsInt64("idx"))
	isSitemap := ctx.Params("idx") != ""
	if opts.Page <= 1 {
		opts.Page = ctx.FormInt("page")
	}
	if opts.Page <= 1 {
		opts.Page = 1
	}

	if isSitemap {
		opts.PageSize = setting.UI.SitemapPagingNum
	}

	var (
		users []*user_model.User
		count int64
		err   error
	)

	sortOrder := ctx.FormString("sort")
	if sortOrder == "" {
		sortOrder = setting.UI.ExploreDefaultSort
	}
	ctx.Data["SortType"] = sortOrder

	orderBy := MapSortOrder(sortOrder)

	if orderBy == "" {
		// In case the `sortType` is not valid, we set it to `recentupdate`.
		sortOrder = "recentupdate"
		ctx.Data["SortType"] = "recentupdate"
		orderBy = MapSortOrder(sortOrder)
	}

	if opts.SupportedSortOrders != nil && !opts.SupportedSortOrders.Contains(sortOrder) {
		ctx.NotFound("unsupported sort order", nil)
		return
	}

	opts.Keyword = ctx.FormTrim("q")
	opts.OrderBy = orderBy
	if len(opts.Keyword) == 0 || isKeywordValid(opts.Keyword) {
		users, count, err = user_model.SearchUsers(ctx, opts)
		if err != nil {
			ctx.ServerError("SearchUsers", err)
			return
		}
	}
	if isSitemap {
		m := sitemap.NewSitemap()
		for _, item := range users {
			m.Add(sitemap.URL{URL: item.HTMLURL(), LastMod: item.UpdatedUnix.AsTimePtr()})
		}
		ctx.Resp.Header().Set("Content-Type", "text/xml")
		if _, err := m.WriteTo(ctx.Resp); err != nil {
			log.Error("Failed writing sitemap: %v", err)
		}
		return
	}

	ctx.Data["Keyword"] = opts.Keyword
	ctx.Data["Total"] = count
	ctx.Data["Users"] = users
	if opts.Load2FAStatus {
		ctx.Data["UsersTwoFaStatus"] = user_model.UserList(users).GetTwoFaStatus(ctx)
	}
	ctx.Data["ShowUserEmail"] = setting.UI.ShowUserEmail
	ctx.Data["IsRepoIndexerEnabled"] = setting.Indexer.RepoIndexerEnabled

	pager := context.NewPagination(int(count), opts.PageSize, opts.Page, 5)
	pager.SetDefaultParams(ctx)
	for paramKey, paramVal := range opts.ExtraParamStrings {
		pager.AddParamString(paramKey, paramVal)
	}
	ctx.Data["Page"] = pager

	ctx.HTML(http.StatusOK, tplName)
}

// Users render explore users page.
func Users(ctx *context.Context) {
	if setting.Service.Explore.DisableUsersPage {
		ctx.Redirect(setting.AppSubURL + "/explore")
		return
	}
	ctx.Data["OrganizationsPageIsDisabled"] = setting.Service.Explore.DisableOrganizationsPage
	ctx.Data["CodePageIsDisabled"] = setting.Service.Explore.DisableCodePage
	ctx.Data["Title"] = ctx.Tr("explore")
	ctx.Data["PageIsExplore"] = true
	ctx.Data["PageIsExploreUsers"] = true
	ctx.Data["IsRepoIndexerEnabled"] = setting.Indexer.RepoIndexerEnabled

	supportedSortOrders := container.SetOf(
		"newest",
		"oldest",
		"alphabetically",
		"reversealphabetically",
	)
	sortOrder := ctx.FormString("sort")
	if sortOrder == "" {
		if supportedSortOrders.Contains(setting.UI.ExploreDefaultSort) {
			sortOrder = setting.UI.ExploreDefaultSort
		} else {
			sortOrder = "newest"
		}
		ctx.SetFormString("sort", sortOrder)
	}

	RenderUserSearch(ctx, &user_model.SearchUserOptions{
		Actor:       ctx.Doer,
		Type:        user_model.UserTypeIndividual,
		ListOptions: db.ListOptions{PageSize: setting.UI.ExplorePagingNum},
		IsActive:    optional.Some(true),
		Visible:     []structs.VisibleType{structs.VisibleTypePublic, structs.VisibleTypeLimited, structs.VisibleTypePrivate},

		SupportedSortOrders: supportedSortOrders,
	}, tplExploreUsers)
}

// Maps a sort query to a database search order.
//
// We cannot use `models.SearchOrderByXxx`, because there may be a JOIN in the statement, different tables may have the same name columns.
func MapSortOrder(sortOrder string) db.SearchOrderBy {
	switch sortOrder {
	case "newest":
		return "`user`.created_unix DESC"

	case "oldest":
		return "`user`.created_unix ASC"

	case "leastupdate":
		return "`user`.updated_unix ASC"

	case "reversealphabetically":
		return "`user`.name DESC"

	case "lastlogin":
		return "`user`.last_login_unix ASC"

	case "reverselastlogin":
		return "`user`.last_login_unix DESC"

	case "alphabetically":
		return "`user`.name ASC"

	case "recentupdate":
		return "`user`.updated_unix DESC"

	default:
		return ""
	}
}
