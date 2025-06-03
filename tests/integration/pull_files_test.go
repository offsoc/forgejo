// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"strings"
	"testing"
	"time"

	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	"forgejo.org/modules/git"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPullFilesCommitHeader(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	t.Run("Verify commit info", func(t *testing.T) {
		repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
		gitRepo, err := git.OpenRepository(git.DefaultContext, repo.RepoPath())
		require.NoError(t, err)
		defer gitRepo.Close()

		commit, err := gitRepo.GetCommit("62fb502a7172d4453f0322a2cc85bddffa57f07a")
		require.NoError(t, err)

		req := NewRequest(t, "GET", "/user2/repo1/pulls/5/commits/62fb502a7172d4453f0322a2cc85bddffa57f07a")
		resp := MakeRequest(t, req, http.StatusOK)

		htmlDoc := NewHTMLParser(t, resp.Body)
		header := htmlDoc.doc.Find("#diff-commit-header")

		summary := header.Find(".commit-header h3")
		assert.Equal(t, commit.Summary(), strings.TrimSpace(summary.Text()))

		author := header.Find(".author strong")
		assert.Equal(t, commit.Author.Name, author.Text())

		date, _ := header.Find("#authored-time relative-time").Attr("datetime")
		assert.Equal(t, commit.Author.When.Format(time.RFC3339), date)

		sha := header.Find(".commit-header-row .sha.label")
		shaHref, _ := sha.Attr("href")
		assert.Equal(t, commit.ID.String()[:10], sha.Find(".shortsha").Text())
		assert.Equal(t, "/user2/repo1/commit/62fb502a7172d4453f0322a2cc85bddffa57f07a", shaHref)
	})

	t.Run("Navigation", func(t *testing.T) {
		t.Run("No previous on first commit", func(t *testing.T) {
			req := NewRequest(t, "GET", "/user2/commitsonpr/pulls/1/commits/4ca8bcaf27e28504df7bf996819665986b01c847")
			resp := MakeRequest(t, req, http.StatusOK)

			htmlDoc := NewHTMLParser(t, resp.Body)
			buttons := htmlDoc.doc.Find(".commit-header-buttons a.tiny.button")

			assert.Equal(t, 2, buttons.Length(), "expected two buttons in commit header")

			assert.True(t, buttons.First().HasClass("disabled"), "'prev' button should be disabled")
			assert.False(t, buttons.Last().HasClass("disabled"), "'next' button should not be disabled")

			href, _ := buttons.Last().Attr("href")
			assert.Equal(t, "/user2/commitsonpr/pulls/1/commits/96cef4a7b72b3c208340ae6f0cf55a93e9077c93", href)
		})

		t.Run("No next on last commit", func(t *testing.T) {
			req := NewRequest(t, "GET", "/user2/commitsonpr/pulls/1/commits/1978192d98bb1b65e11c2cf37da854fbf94bffd6")
			resp := MakeRequest(t, req, http.StatusOK)

			htmlDoc := NewHTMLParser(t, resp.Body)
			buttons := htmlDoc.doc.Find(".commit-header-buttons a.tiny.button")

			assert.Equal(t, 2, buttons.Length(), "expected two buttons in commit header")

			assert.False(t, buttons.First().HasClass("disabled"), "'prev' button should not be disabled")
			assert.True(t, buttons.Last().HasClass("disabled"), "'next' button should be disabled")

			href, _ := buttons.First().Attr("href")
			assert.Equal(t, "/user2/commitsonpr/pulls/1/commits/837d5c8125633d7d258f93b998e867eab0145520", href)
		})

		t.Run("Both directions on middle commit", func(t *testing.T) {
			req := NewRequest(t, "GET", "/user2/commitsonpr/pulls/1/commits/c5626fc9eff57eb1bb7b796b01d4d0f2f3f792a2")
			resp := MakeRequest(t, req, http.StatusOK)

			htmlDoc := NewHTMLParser(t, resp.Body)
			buttons := htmlDoc.doc.Find(".commit-header-buttons a.tiny.button")

			assert.Equal(t, 2, buttons.Length(), "expected two buttons in commit header")

			assert.False(t, buttons.First().HasClass("disabled"), "'prev' button should not be disabled")
			assert.False(t, buttons.Last().HasClass("disabled"), "'next' button should not be disabled")

			href, _ := buttons.First().Attr("href")
			assert.Equal(t, "/user2/commitsonpr/pulls/1/commits/96cef4a7b72b3c208340ae6f0cf55a93e9077c93", href)

			href, _ = buttons.Last().Attr("href")
			assert.Equal(t, "/user2/commitsonpr/pulls/1/commits/23576dd018294e476c06e569b6b0f170d0558705", href)
		})
	})
}
