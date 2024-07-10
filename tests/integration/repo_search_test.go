// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"testing"

	"code.gitea.io/gitea/models/db"
	repo_model "code.gitea.io/gitea/models/repo"
	code_indexer "code.gitea.io/gitea/modules/indexer/code"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/test"
	"code.gitea.io/gitea/routers"
	"code.gitea.io/gitea/tests"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
)

func resultFilenames(t testing.TB, doc *HTMLDoc) []string {
	resultSelections := doc.
		Find(".repository.search").
		Find("details.repo-search-result")

	result := make([]string, resultSelections.Length())
	resultSelections.Each(func(i int, selection *goquery.Selection) {
		assert.True(t, resultSelections.Find("div ol li").Length() > 0)
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
	assert.NoError(t, err)

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
	assert.NoError(t, err)

	if indexer {
		code_indexer.UpdateRepoIndexer(repo)
	}

	testSearch(t, "/user2/glob/search?q=loren&page=1", []string{"a.txt"}, indexer)
	testSearch(t, "/user2/glob/search?q=loren&page=1&fuzzy=false", []string{"a.txt"}, indexer)

	if indexer {
		// fuzzy search: matches both file3 (x/b.txt) and file1 (a.txt)
		// when indexer is enabled
		testSearch(t, "/user2/glob/search?q=file3&page=1", []string{"x/b.txt", "a.txt"}, indexer)
		testSearch(t, "/user2/glob/search?q=file4&page=1", []string{"x/b.txt", "a.txt"}, indexer)
		testSearch(t, "/user2/glob/search?q=file5&page=1", []string{"x/b.txt", "a.txt"}, indexer)
	} else {
		// fuzzy search: Union/OR of all the keywords
		// when indexer is disabled
		testSearch(t, "/user2/glob/search?q=file3+file1&page=1", []string{"a.txt", "x/b.txt"}, indexer)
		testSearch(t, "/user2/glob/search?q=file4&page=1", []string{}, indexer)
		testSearch(t, "/user2/glob/search?q=file5&page=1", []string{}, indexer)
	}

	testSearch(t, "/user2/glob/search?q=file3&page=1&fuzzy=false", []string{"x/b.txt"}, indexer)
	testSearch(t, "/user2/glob/search?q=file4&page=1&fuzzy=false", []string{}, indexer)
	testSearch(t, "/user2/glob/search?q=file5&page=1&fuzzy=false", []string{}, indexer)
}

func testSearch(t *testing.T, url string, expected []string, indexer bool) {
	req := NewRequest(t, "GET", url)
	resp := MakeRequest(t, req, http.StatusOK)

	doc := NewHTMLParser(t, resp.Body)
	container := doc.Find(".repository").Find(".ui.container")

	grepMsg := container.Find(".ui.message[data-test-tag=grep]")
	assert.EqualValues(t, indexer, len(grepMsg.Nodes) == 0)

	branchDropdown := container.Find(".js-branch-tag-selector")
	assert.EqualValues(t, indexer, len(branchDropdown.Nodes) == 0)

	// if indexer is disabled "fuzzy" should be displayed as "union"
	expectedFuzzy := "Fuzzy"
	if !indexer {
		expectedFuzzy = "Union"
	}

	fuzzyDropdown := container.Find(".ui.dropdown[data-test-tag=fuzzy-dropdown]")
	actualFuzzyText := fuzzyDropdown.Find(".menu .item[data-value=true]").First().Text()
	assert.EqualValues(t, expectedFuzzy, actualFuzzyText)

	if fuzzyDropdown.
		Find("input[name=fuzzy][value=true]").
		Length() != 0 {
		actualFuzzyText = fuzzyDropdown.Find("div.text").First().Text()
		assert.EqualValues(t, expectedFuzzy, actualFuzzyText)
	}

	filenames := resultFilenames(t, doc)
	assert.EqualValues(t, expected, filenames)
}
