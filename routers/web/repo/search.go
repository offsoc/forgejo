// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"net/http"
	"strings"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/modules/base"
	"code.gitea.io/gitea/modules/git"
	code_indexer "code.gitea.io/gitea/modules/indexer/code"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/services/context"
)

const tplSearch base.TplName = "repo/search"

type searchMode int

const (
	ExactSearchMode searchMode = iota
	FuzzySearchMode
	RegExpSearchMode
)

func searchModeFromString(s string) searchMode {
	switch s {
	case "fuzzy", "union":
		return FuzzySearchMode
	case "regexp":
		return RegExpSearchMode
	default:
		return ExactSearchMode
	}
}

func (m searchMode) String() string {
	switch m {
	case ExactSearchMode:
		return "exact"
	case FuzzySearchMode:
		return "fuzzy"
	case RegExpSearchMode:
		return "regexp"
	default:
		panic("cannot happen")
	}
}

// Search render repository search page
func Search(ctx *context.Context) {
	language := ctx.FormTrim("l")
	keyword := ctx.FormTrim("q")

	mode := ExactSearchMode
	if modeStr := ctx.FormString("mode"); len(modeStr) > 0 {
		mode = searchModeFromString(modeStr)
	} else if ctx.FormOptionalBool("fuzzy").ValueOrDefault(true) { // for backward compatibility in links
		mode = FuzzySearchMode
	}

	ctx.Data["Keyword"] = keyword
	ctx.Data["Language"] = language
	ctx.Data["CodeSearchMode"] = mode.String()
	ctx.Data["PageIsViewCode"] = true

	if keyword == "" {
		ctx.HTML(http.StatusOK, tplSearch)
		return
	}

	page := ctx.FormInt("page")
	if page <= 0 {
		page = 1
	}

	var total int
	var searchResults []*code_indexer.Result
	var searchResultLanguages []*code_indexer.SearchResultLanguages
	if setting.Indexer.RepoIndexerEnabled {
		var err error
		total, searchResults, searchResultLanguages, err = code_indexer.PerformSearch(ctx, &code_indexer.SearchOptions{
			RepoIDs:        []int64{ctx.Repo.Repository.ID},
			Keyword:        keyword,
			IsKeywordFuzzy: mode == FuzzySearchMode,
			Language:       language,
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
		ctx.Data["CodeSearchOptions"] = []string{"exact", "fuzzy"}
	} else {
		grepOpt := git.GrepOptions{
			ContextLineNumber: 1,
			RefName:           ctx.Repo.RefName,
		}
		switch mode {
		case FuzzySearchMode:
			grepOpt.Mode = git.FixedAnyGrepMode
			ctx.Data["CodeSearchMode"] = "union"
		case RegExpSearchMode:
			grepOpt.Mode = git.RegExpGrepMode
		}
		res, err := git.GrepSearch(ctx, ctx.Repo.GitRepo, keyword, grepOpt)
		if err != nil {
			ctx.ServerError("GrepSearch", err)
			return
		}
		total = len(res)
		pageStart := min((page-1)*setting.UI.RepoSearchPagingNum, len(res))
		pageEnd := min(page*setting.UI.RepoSearchPagingNum, len(res))
		res = res[pageStart:pageEnd]
		for _, r := range res {
			searchResults = append(searchResults, &code_indexer.Result{
				RepoID:   ctx.Repo.Repository.ID,
				Filename: r.Filename,
				CommitID: ctx.Repo.CommitID,
				// UpdatedUnix: not supported yet
				// Language:    not supported yet
				// Color:       not supported yet
				Lines: code_indexer.HighlightSearchResultCode(r.Filename, r.LineNumbers, r.HighlightedRanges, strings.Join(r.LineCodes, "\n")),
			})
		}
		ctx.Data["CodeSearchOptions"] = []string{"exact", "union", "regexp"}
	}

	ctx.Data["CodeIndexerDisabled"] = !setting.Indexer.RepoIndexerEnabled
	ctx.Data["Repo"] = ctx.Repo.Repository
	ctx.Data["SourcePath"] = ctx.Repo.Repository.Link()
	ctx.Data["SearchResults"] = searchResults
	ctx.Data["SearchResultLanguages"] = searchResultLanguages

	pager := context.NewPagination(total, setting.UI.RepoSearchPagingNum, page, 5)
	pager.SetDefaultParams(ctx)
	pager.AddParam(ctx, "l", "Language")
	ctx.Data["Page"] = pager

	ctx.HTML(http.StatusOK, tplSearch)
}
