// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"testing"

	"code.gitea.io/gitea/models/db"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/unit"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestViewPulls(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	req := NewRequest(t, "GET", "/user2/repo1/pulls")
	resp := MakeRequest(t, req, http.StatusOK)

	htmlDoc := NewHTMLParser(t, resp.Body)
	search := htmlDoc.doc.Find(".list-header-search > .search > .input > input")
	placeholder, _ := search.Attr("placeholder")
	assert.Equal(t, "Search pulls...", placeholder)
}

func TestPullManuallyMergeWarning(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	session := loginUser(t, user2.Name)

	warningMessage := `Warning: The "Autodetect manual merge" setting is not enabled for this repository, you will have to mark this pull request as manually merged afterwards.`
	t.Run("Autodetect disabled", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, "GET", "/user2/repo1/pulls/3")
		resp := session.MakeRequest(t, req, http.StatusOK)

		htmlDoc := NewHTMLParser(t, resp.Body)
		mergeInstructions := htmlDoc.Find("#merge-instructions").Text()
		assert.Contains(t, mergeInstructions, warningMessage)
	})

	pullRequestUnit := unittest.AssertExistsAndLoadBean(t, &repo_model.RepoUnit{RepoID: 1, Type: unit.TypePullRequests})
	config := pullRequestUnit.PullRequestsConfig()
	config.AutodetectManualMerge = true
	_, err := db.GetEngine(db.DefaultContext).ID(pullRequestUnit.ID).Cols("config").Update(pullRequestUnit)
	require.NoError(t, err)

	t.Run("Autodetect enabled", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, "GET", "/user2/repo1/pulls/3")
		resp := session.MakeRequest(t, req, http.StatusOK)

		htmlDoc := NewHTMLParser(t, resp.Body)
		mergeInstructions := htmlDoc.Find("#merge-instructions").Text()
		assert.NotContains(t, mergeInstructions, warningMessage)
	})
}

func TestViewPullRequest(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	req := NewRequest(t, "GET", "/user2/repo1/pulls/3")
	resp := MakeRequest(t, req, http.StatusOK)

	htmlDoc := NewHTMLParser(t, resp.Body)

	t.Run("DownloadLinks", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		downloadLinks := htmlDoc.doc.Find(".ui.download > div.tw-mt-2 > a")
		assert.Equal(t, 3, downloadLinks.Length())

		patchDownloadLink := downloadLinks.First()
		patchHref, patchExists := patchDownloadLink.Attr("href")
		assert.Equal(t, "/user2/repo1/pulls/3.patch", patchHref)
		assert.True(t, patchExists)
		MakeRequest(t, NewRequest(t, "GET", patchHref), http.StatusOK)

		diffDownloadLink := patchDownloadLink.Next()
		diffHref, diffExists := diffDownloadLink.Attr("href")
		assert.Equal(t, "/user2/repo1/pulls/3.diff", diffHref)
		assert.True(t, diffExists)
		MakeRequest(t, NewRequest(t, "GET", diffHref), http.StatusOK)

		binaryDiffDownloadLink := diffDownloadLink.Next()
		binaryDiffHref, binaryDiffExists := binaryDiffDownloadLink.Attr("href")
		assert.Equal(t, "/user2/repo1/pulls/3.diff?binary=1", binaryDiffHref)
		assert.True(t, binaryDiffExists)
		MakeRequest(t, NewRequest(t, "GET", binaryDiffHref), http.StatusOK)
	})
}
