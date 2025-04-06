// Copyright 2024 The Forgejo Authors c/o Codeberg e.V.. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"testing"

	auth_model "forgejo.org/models/auth"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	"forgejo.org/modules/setting"
	api "forgejo.org/modules/structs"
	"forgejo.org/tests"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
)

func TestWikiSearchContent(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	req := NewRequest(t, "GET", "/user2/repo1/wiki/search?q=This")
	resp := MakeRequest(t, req, http.StatusOK)
	doc := NewHTMLParser(t, resp.Body)
	res := doc.Find(".item > b").Map(func(_ int, el *goquery.Selection) string {
		return el.Text()
	})
	assert.Equal(t, []string{
		"Home",
		"Page With Spaced Name",
		"Unescaped File",
	}, res)
}

func TestWikiBranchNormalize(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	username := "user2"
	session := loginUser(t, username)
	settingsURLStr := "/user2/repo1/settings"

	assertNormalizeButton := func(present bool) string {
		req := NewRequest(t, "GET", settingsURLStr) //.AddTokenAuth(token)
		resp := session.MakeRequest(t, req, http.StatusOK)
		htmlDoc := NewHTMLParser(t, resp.Body)
		htmlDoc.AssertElement(t, "button[data-modal='#rename-wiki-branch-modal']", present)

		return htmlDoc.GetCSRF()
	}

	// By default the repo wiki branch is empty
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	assert.Empty(t, repo.WikiBranch)

	// This means we default to setting.Repository.DefaultBranch
	assert.Equal(t, setting.Repository.DefaultBranch, repo.GetWikiBranchName())

	// Which further means that the "Normalize wiki branch" parts do not appear on settings
	assertNormalizeButton(false)

	// Lets rename the branch!
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository)
	repoURLStr := fmt.Sprintf("/api/v1/repos/%s/%s", username, repo.Name)
	wikiBranch := "wiki"
	req := NewRequestWithJSON(t, "PATCH", repoURLStr, &api.EditRepoOption{
		WikiBranch: &wikiBranch,
	}).AddTokenAuth(token)
	MakeRequest(t, req, http.StatusOK)

	// The wiki branch should now be changed
	repo = unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	assert.Equal(t, wikiBranch, repo.GetWikiBranchName())

	// And as such, the button appears!
	csrf := assertNormalizeButton(true)

	// Invoking the normalization renames the wiki branch back to the default
	req = NewRequestWithValues(t, "POST", settingsURLStr, map[string]string{
		"_csrf":     csrf,
		"action":    "rename-wiki-branch",
		"repo_name": repo.FullName(),
	})
	session.MakeRequest(t, req, http.StatusSeeOther)

	repo = unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	assert.Equal(t, setting.Repository.DefaultBranch, repo.GetWikiBranchName())
	assertNormalizeButton(false)
}

func TestWikiTOC(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	username := "user2"
	session := loginUser(t, username)

	t.Run("Link in heading", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequestWithValues(t, "POST", "/user2/repo1/wiki/Home?action=_edit", map[string]string{
			"_csrf":   GetCSRF(t, session, "/user2/repo1/wiki/Home"),
			"title":   "Home",
			"content": "# [Helpdesk](Helpdesk)",
		})
		session.MakeRequest(t, req, http.StatusSeeOther)

		req = NewRequest(t, "GET", "/user2/repo1/wiki/Home")
		resp := MakeRequest(t, req, http.StatusOK)
		htmlDoc := NewHTMLParser(t, resp.Body)

		assert.Equal(t, "Helpdesk", htmlDoc.Find(".wiki-content-toc a").Text())
	})
}
