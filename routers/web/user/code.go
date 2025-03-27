// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user

import (
	"net/http"

	"forgejo.org/models/db"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/modules/base"
	code_indexer "forgejo.org/modules/indexer/code"
	"forgejo.org/modules/setting"
	shared_user "forgejo.org/routers/web/shared/user"
	"forgejo.org/services/context"
)

const (
	tplUserCode base.TplName = "user/code"
)

// CodeSearch render user/organization code search page
func CodeSearch(ctx *context.Context) {
	if !setting.Indexer.RepoIndexerEnabled {
		ctx.Redirect(ctx.ContextUser.HomeLink())
		return
	}
	shared_user.PrepareContextForProfileBigAvatar(ctx)
	shared_user.RenderUserHeader(ctx)

	if err := shared_user.LoadHeaderCount(ctx); err != nil {
		ctx.ServerError("LoadHeaderCount", err)
		return
	}

	ctx.Data["IsPackageEnabled"] = setting.Packages.Enabled
	ctx.Data["IsRepoIndexerEnabled"] = setting.Indexer.RepoIndexerEnabled
	ctx.Data["Title"] = ctx.Tr("explore.code")

	language := ctx.FormTrim("l")
	keyword := ctx.FormTrim("q")
	path := ctx.FormTrim("path")

	mode := code_indexer.SearchModeExact
	if m := ctx.FormTrim("mode"); m == "union" ||
		m == "fuzzy" ||
		ctx.FormBool("fuzzy") {
		mode = code_indexer.SearchModeUnion
	}

	ctx.Data["Keyword"] = keyword
	ctx.Data["Language"] = language
	ctx.Data["CodeSearchOptions"] = code_indexer.CodeSearchOptions
	ctx.Data["CodeSearchMode"] = mode.String()
	ctx.Data["IsCodePage"] = true

	if keyword == "" {
		ctx.HTML(http.StatusOK, tplUserCode)
		return
	}

	var (
		repoIDs []int64
		err     error
	)

	page := ctx.FormInt("page")
	if page <= 0 {
		page = 1
	}

	repoIDs, err = repo_model.FindUserCodeAccessibleOwnerRepoIDs(ctx, ctx.ContextUser.ID, ctx.Doer)
	if err != nil {
		ctx.ServerError("FindUserCodeAccessibleOwnerRepoIDs", err)
		return
	}

	var (
		total                 int
		searchResults         []*code_indexer.Result
		searchResultLanguages []*code_indexer.SearchResultLanguages
	)

	if len(repoIDs) > 0 {
		total, searchResults, searchResultLanguages, err = code_indexer.PerformSearch(ctx, &code_indexer.SearchOptions{
			RepoIDs:  repoIDs,
			Keyword:  keyword,
			Mode:     mode,
			Language: language,
			Filename: path,
			Paginator: &db.ListOptions{
				Page:     page,
				PageSize: setting.UI.RepoSearchPagingNum,
			},
		})
		if err != nil {
			if code_indexer.IsAvailable(ctx) {
				ctx.ServerError("SearchResults", err)
				return
			}
			ctx.Data["CodeIndexerUnavailable"] = true
		} else {
			ctx.Data["CodeIndexerUnavailable"] = !code_indexer.IsAvailable(ctx)
		}

		loadRepoIDs := make([]int64, 0, len(searchResults))
		for _, result := range searchResults {
			var find bool
			for _, id := range loadRepoIDs {
				if id == result.RepoID {
					find = true
					break
				}
			}
			if !find {
				loadRepoIDs = append(loadRepoIDs, result.RepoID)
			}
		}

		repoMaps, err := repo_model.GetRepositoriesMapByIDs(ctx, loadRepoIDs)
		if err != nil {
			ctx.ServerError("GetRepositoriesMapByIDs", err)
			return
		}

		ctx.Data["RepoMaps"] = repoMaps
	}
	ctx.Data["SearchResults"] = searchResults
	ctx.Data["SearchResultLanguages"] = searchResultLanguages

	pager := context.NewPagination(total, setting.UI.RepoSearchPagingNum, page, 5)
	pager.SetDefaultParams(ctx)
	pager.AddParam(ctx, "l", "Language")
	ctx.Data["Page"] = pager

	ctx.HTML(http.StatusOK, tplUserCode)
}
