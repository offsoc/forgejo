// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"net/http"
	"strings"

	"forgejo.org/models/db"
	"forgejo.org/modules/base"
	"forgejo.org/modules/git"
	code_indexer "forgejo.org/modules/indexer/code"
	"forgejo.org/modules/setting"
	"forgejo.org/services/context"
)

const tplSearch base.TplName = "repo/search"

type searchMode int

const (
	ExactSearchMode searchMode = iota
	UnionSearchMode
	RegExpSearchMode
)

func searchModeFromString(s string) searchMode {
	switch s {
	case "fuzzy", "union":
		return UnionSearchMode
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
	case UnionSearchMode:
		return "union"
	case RegExpSearchMode:
		return "regexp"
	default:
		panic("cannot happen")
	}
}

func (m searchMode) ToIndexer() code_indexer.SearchMode {
	if m == ExactSearchMode {
		return code_indexer.SearchModeExact
	}
	return code_indexer.SearchModeUnion
}

func (m searchMode) ToGitGrep() git.GrepMode {
	switch m {
	case RegExpSearchMode:
		return git.RegExpGrepMode
	case UnionSearchMode:
		return git.FixedAnyGrepMode
	default:
		return git.FixedGrepMode
	}
}

// Search render repository search page
func Search(ctx *context.Context) {
	language := ctx.FormTrim("l")
	keyword := ctx.FormTrim("q")

	path := ctx.FormTrim("path")
	mode := ExactSearchMode
	if modeStr := ctx.FormString("mode"); len(modeStr) > 0 {
		mode = searchModeFromString(modeStr)
	} else if ctx.FormOptionalBool("fuzzy").ValueOrDefault(true) { // for backward compatibility in links
		mode = UnionSearchMode
	}

	ctx.Data["Keyword"] = keyword
	ctx.Data["Language"] = language
	ctx.Data["CodeSearchPath"] = path
	ctx.Data["CodeSearchMode"] = mode.String()
	ctx.Data["PageIsViewCode"] = true
	ctx.Data["CodeIndexerDisabled"] = !setting.Indexer.RepoIndexerEnabled
	if setting.Indexer.RepoIndexerEnabled {
		ctx.Data["CodeSearchOptions"] = code_indexer.CodeSearchOptions
	} else {
		ctx.Data["CodeSearchOptions"] = git.GrepSearchOptions
	}

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
			RepoIDs:  []int64{ctx.Repo.Repository.ID},
			Keyword:  keyword,
			Mode:     mode.ToIndexer(),
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
	} else {
		res, err := git.GrepSearch(ctx, ctx.Repo.GitRepo, keyword, git.GrepOptions{
			ContextLineNumber: 1,
			RefName:           ctx.Repo.RefName,
			Filename:          path,
			Mode:              mode.ToGitGrep(),
		})
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
				Lines: code_indexer.HighlightSearchResultCode(
					r.Filename, r.LineNumbers, r.HighlightedRanges,
					strings.Join(r.LineCodes, "\n")),
			})
		}
	}

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
