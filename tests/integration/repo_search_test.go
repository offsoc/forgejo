// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"testing"

	"forgejo.org/models/db"
	repo_model "forgejo.org/models/repo"
	code_indexer "forgejo.org/modules/indexer/code"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/test"
	"forgejo.org/routers"
	"forgejo.org/tests"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resultFilenames(t testing.TB, doc *HTMLDoc) []string {
	resultSelections := doc.
		Find(".repository.search").
		Find("details.repo-search-result")

	result := make([]string, resultSelections.Length())
	resultSelections.Each(func(i int, selection *goquery.Selection) {
		assert.Positive(t, selection.Find("div ol li").Length(), 0)
		assert.Positive(t, selection.Find(".code-inner").Find(".search-highlight").Length(), 0)
		result[i] = selection.
			Find(".header").
			Find("span.file a.file-link").
			First().
			Text()
	})

	return result
}

func TestSearchRepoIndexer(t *testing.T) {
	testSearchRepo(t, true)
}

func TestSearchRepoNoIndexer(t *testing.T) {
	testSearchRepo(t, false)
}

func testSearchRepo(t *testing.T, indexer bool) {
	defer tests.PrepareTestEnv(t)()
	defer test.MockVariableValue(&setting.Indexer.RepoIndexerEnabled, indexer)()
	defer test.MockVariableValue(&testWebRoutes, routers.NormalRoutes())()

	repo, err := repo_model.GetRepositoryByOwnerAndName(db.DefaultContext, "user2", "repo1")
	require.NoError(t, err)

	if indexer {
		code_indexer.UpdateRepoIndexer(repo)
	}

	testSearch(t, "/user2/repo1/search?q=Description&page=1", []string{"README.md"}, indexer)

	req := NewRequest(t, "HEAD", "/user2/repo1/search/branch/"+repo.DefaultBranch)
	if indexer {
		MakeRequest(t, req, http.StatusNotFound)
	} else {
		MakeRequest(t, req, http.StatusOK)
	}

	defer test.MockVariableValue(&setting.Indexer.IncludePatterns, setting.IndexerGlobFromString("**.txt"))()
	defer test.MockVariableValue(&setting.Indexer.ExcludePatterns, setting.IndexerGlobFromString("**/y/**"))()

	repo, err = repo_model.GetRepositoryByOwnerAndName(db.DefaultContext, "user2", "glob")
	require.NoError(t, err)

	if indexer {
		code_indexer.UpdateRepoIndexer(repo)
	}

	testSearch(t, "/user2/glob/search?q=", []string{}, indexer)
	testSearch(t, "/user2/glob/search?q=loren&page=1", []string{"a.txt"}, indexer)
	testSearch(t, "/user2/glob/search?q=loren&page=1&mode=exact", []string{"a.txt"}, indexer)

	// union search: Union/OR of all the keywords
	testSearch(t, "/user2/glob/search?q=file3+file1&mode=union&page=1", []string{"a.txt", "x/b.txt"}, indexer)
	testSearch(t, "/user2/glob/search?q=file4&mode=union&page=1", []string{}, indexer)
	testSearch(t, "/user2/glob/search?q=file5&mode=union&page=1", []string{}, indexer)

	testSearch(t, "/user2/glob/search?q=file3&page=1&mode=exact", []string{"x/b.txt"}, indexer)
	testSearch(t, "/user2/glob/search?q=file4&page=1&mode=exact", []string{}, indexer)
	testSearch(t, "/user2/glob/search?q=file5&page=1&mode=exact", []string{}, indexer)
}

func testSearch(t *testing.T, url string, expected []string, indexer bool) {
	req := NewRequest(t, "GET", url)
	resp := MakeRequest(t, req, http.StatusOK)

	doc := NewHTMLParser(t, resp.Body)
	container := doc.Find(".repository").Find(".ui.container")

	branchDropdown := container.Find(".js-branch-tag-selector")
	assert.Equal(t, indexer, len(branchDropdown.Nodes) == 0)

	dropdownOptions := container.
		Find(".menu[data-test-tag=fuzzy-dropdown]").
		Find("input[type=radio][name=mode]").
		Map(func(_ int, sel *goquery.Selection) string {
			attr, exists := sel.Attr("value")
			assert.True(t, exists)
			return attr
		})

	if indexer {
		assert.Equal(t, []string{"exact", "union"}, dropdownOptions)
	} else {
		assert.Equal(t, []string{"exact", "union", "regexp"}, dropdownOptions)
	}

	filenames := resultFilenames(t, doc)
	assert.ElementsMatch(t, expected, filenames)
}
