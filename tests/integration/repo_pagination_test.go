// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"path"
	"testing"

	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/test"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
)

func TestRepoPaginations(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	t.Run("Fork", func(t *testing.T) {
		// Make forks of user2/repo1
		session := loginUser(t, "user2")
		testRepoFork(t, session, "user2", "repo1", "org3", "repo1")
		session = loginUser(t, "user5")
		testRepoFork(t, session, "user2", "repo1", "org6", "repo1")

		unittest.AssertCount(t, &repo_model.Repository{ForkID: 1}, 2)

		testRepoPagination(t, session, "user2/repo1", "forks", &setting.MaxForksPerPage)
	})
	t.Run("Stars", func(t *testing.T) {
		// Add stars to user2/repo1.
		session := loginUser(t, "user2")
		req := NewRequestWithValues(t, "POST", "/user2/repo1/action/star", map[string]string{
			"_csrf": GetCSRF(t, session, "/user2/repo1"),
		})
		session.MakeRequest(t, req, http.StatusOK)

		session = loginUser(t, "user1")
		req = NewRequestWithValues(t, "POST", "/user2/repo1/action/star", map[string]string{
			"_csrf": GetCSRF(t, session, "/user2/repo1"),
		})
		session.MakeRequest(t, req, http.StatusOK)

		testRepoPagination(t, session, "user2/repo1", "stars", &setting.MaxUserCardsPerPage)
	})
	t.Run("Watcher", func(t *testing.T) {
		// user2/repo2 is watched by its creator user2. Watch it by user1 to make it watched by 2 users.
		session := loginUser(t, "user1")
		req := NewRequestWithValues(t, "POST", "/user2/repo2/action/watch", map[string]string{
			"_csrf": GetCSRF(t, session, "/user2/repo2"),
		})
		session.MakeRequest(t, req, http.StatusOK)

		testRepoPagination(t, session, "user2/repo2", "watchers", &setting.MaxUserCardsPerPage)
	})
}

func testRepoPagination(t *testing.T, session *TestSession, repo, kind string, mockableVar *int) {
	t.Run("Should paginate", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		defer test.MockVariableValue(mockableVar, 1)()
		req := NewRequest(t, "GET", "/"+path.Join(repo, kind))
		resp := session.MakeRequest(t, req, http.StatusOK)
		htmlDoc := NewHTMLParser(t, resp.Body)

		paginationButton := htmlDoc.Find(".item.navigation[href='/" + path.Join(repo, kind) + "?page=2']")
		// Next and Last button.
		assert.Equal(t, 2, paginationButton.Length())
	})

	t.Run("Shouldn't paginate", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		defer test.MockVariableValue(mockableVar, 2)()
		req := NewRequest(t, "GET", "/"+path.Join(repo, kind))
		resp := session.MakeRequest(t, req, http.StatusOK)
		htmlDoc := NewHTMLParser(t, resp.Body)

		htmlDoc.AssertElement(t, ".item.navigation[href='/"+path.Join(repo, kind)+"?page=2']", false)
	})
}
