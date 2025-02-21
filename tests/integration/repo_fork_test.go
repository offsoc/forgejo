// Copyright 2017 The Gitea Authors. All rights reserved.
// Copyright 2024 The Forgejo Authors c/o Codeberg e.V.. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"code.gitea.io/gitea/models/db"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/modules/test"
	"code.gitea.io/gitea/routers"
	repo_service "code.gitea.io/gitea/services/repository"
	"code.gitea.io/gitea/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testRepoFork(t *testing.T, session *TestSession, ownerName, repoName, forkOwnerName, forkRepoName string) *httptest.ResponseRecorder {
	t.Helper()

	forkOwner := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: forkOwnerName})

	// Step0: check the existence of the to-fork repo
	req := NewRequestf(t, "GET", "/%s/%s", forkOwnerName, forkRepoName)
	session.MakeRequest(t, req, http.StatusNotFound)

	// Step1: visit the /fork page
	forkURL := fmt.Sprintf("/%s/%s/fork", ownerName, repoName)
	req = NewRequest(t, "GET", forkURL)
	resp := session.MakeRequest(t, req, http.StatusOK)

	// Step2: fill the form of the forking
	htmlDoc := NewHTMLParser(t, resp.Body)
	link, exists := htmlDoc.doc.Find(fmt.Sprintf("form.ui.form[action=\"%s\"]", forkURL)).Attr("action")
	assert.True(t, exists, "The template has changed")
	_, exists = htmlDoc.doc.Find(fmt.Sprintf(".owner.dropdown .item[data-value=\"%d\"]", forkOwner.ID)).Attr("data-value")
	assert.True(t, exists, "Fork owner %q is not present in select box", forkOwnerName)
	req = NewRequestWithValues(t, "POST", link, map[string]string{
		"_csrf":     htmlDoc.GetCSRF(),
		"uid":       fmt.Sprintf("%d", forkOwner.ID),
		"repo_name": forkRepoName,
	})
	session.MakeRequest(t, req, http.StatusSeeOther)

	// Step3: check the existence of the forked repo
	req = NewRequestf(t, "GET", "/%s/%s", forkOwnerName, forkRepoName)
	resp = session.MakeRequest(t, req, http.StatusOK)

	return resp
}

func testRepoForkLegacyRedirect(t *testing.T, session *TestSession, ownerName, repoName string) {
	t.Helper()

	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: ownerName})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{OwnerID: owner.ID, Name: repoName})

	// Visit the /repo/fork/:id url
	req := NewRequestf(t, "GET", "/repo/fork/%d", repo.ID)
	resp := session.MakeRequest(t, req, http.StatusMovedPermanently)

	assert.Equal(t, repo.Link()+"/fork", resp.Header().Get("Location"))
}

func TestRepoFork(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		user5 := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: "user5"})
		session := loginUser(t, user5.Name)

		t.Run("by name", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			defer func() {
				repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{OwnerID: user5.ID, Name: "repo1"})
				repo_service.DeleteRepository(db.DefaultContext, user5, repo, false)
			}()
			testRepoFork(t, session, "user2", "repo1", "user5", "repo1")
		})

		t.Run("legacy redirect", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			testRepoForkLegacyRedirect(t, session, "user2", "repo1")

			t.Run("private 404", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				// Make sure the repo we try to fork is private
				repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 31, IsPrivate: true})

				// user5 does not have access to user2/repo20
				req := NewRequestf(t, "GET", "/repo/fork/%d", repo.ID) // user2/repo20
				session.MakeRequest(t, req, http.StatusNotFound)
			})
			t.Run("authenticated private redirect", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				// Make sure the repo we try to fork is private
				repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 31, IsPrivate: true})

				// user1 has access to user2/repo20
				session := loginUser(t, "user1")
				req := NewRequestf(t, "GET", "/repo/fork/%d", repo.ID) // user2/repo20
				session.MakeRequest(t, req, http.StatusMovedPermanently)
			})
			t.Run("no code unit", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				// Make sure the repo we try to fork is private.
				// We're also choosing user15/big_test_private_2, because it has the Code unit disabled.
				repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 20, IsPrivate: true})

				// user1, even though an admin, can't fork a repo without a code unit.
				session := loginUser(t, "user1")
				req := NewRequestf(t, "GET", "/repo/fork/%d", repo.ID) // user15/big_test_private_2
				session.MakeRequest(t, req, http.StatusNotFound)
			})
		})

		t.Run("fork button", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequest(t, "GET", "/user2/repo1/issues")
			resp := MakeRequest(t, req, http.StatusOK)
			htmlDoc := NewHTMLParser(t, resp.Body)

			forkButton := htmlDoc.Find("a[href*='/forks']")
			assert.EqualValues(t, 1, forkButton.Length())

			href, _ := forkButton.Attr("href")
			assert.Equal(t, "/user2/repo1/forks", href)
			assert.Equal(t, "0", strings.TrimSpace(forkButton.Text()))

			t.Run("no fork button on empty repo", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				// Create an empty repository
				repo, err := repo_service.CreateRepository(db.DefaultContext, user5, user5, repo_service.CreateRepoOptions{
					Name:     "empty-repo",
					AutoInit: false,
				})
				defer func() {
					repo_service.DeleteRepository(db.DefaultContext, user5, repo, false)
				}()
				require.NoError(t, err)
				assert.NotEmpty(t, repo)

				// Load the repository home view
				req := NewRequest(t, "GET", repo.HTMLURL())
				resp := session.MakeRequest(t, req, http.StatusOK)
				htmlDoc := NewHTMLParser(t, resp.Body)

				// On an empty repo, the fork button is not present
				htmlDoc.AssertElement(t, ".basic.button[href*='/fork']", false)
			})
		})

		t.Run("DISABLE_FORKS", func(t *testing.T) {
			defer test.MockVariableValue(&setting.Repository.DisableForks, true)()
			defer test.MockVariableValue(&testWebRoutes, routers.NormalRoutes())()

			t.Run("fork button not present", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				// The "Fork" button should not appear on the repo home
				req := NewRequest(t, "GET", "/user2/repo1")
				resp := MakeRequest(t, req, http.StatusOK)
				htmlDoc := NewHTMLParser(t, resp.Body)
				htmlDoc.AssertElement(t, "[href=/user2/repo1/fork]", false)
			})

			t.Run("forking by URL", func(t *testing.T) {
				t.Run("by name", func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()

					// Forking by URL should be Not Found
					req := NewRequest(t, "GET", "/user2/repo1/fork")
					session.MakeRequest(t, req, http.StatusNotFound)
				})

				t.Run("by legacy URL", func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()

					// Forking by legacy URL should be Not Found
					repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1}) // user2/repo1
					req := NewRequestf(t, "GET", "/repo/fork/%d", repo.ID)
					session.MakeRequest(t, req, http.StatusNotFound)
				})
			})

			t.Run("fork listing", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				// Listing the forks should be Not Found, too
				req := NewRequest(t, "GET", "/user2/repo1/forks")
				MakeRequest(t, req, http.StatusNotFound)
			})
		})
	})
}

func TestRepoForkToOrg(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		session := loginUser(t, "user2")
		org3 := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: "org3"})

		t.Run("by name", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			defer func() {
				repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{OwnerID: org3.ID, Name: "repo1"})
				repo_service.DeleteRepository(db.DefaultContext, org3, repo, false)
			}()

			testRepoFork(t, session, "user2", "repo1", "org3", "repo1")

			// Check that no more forking is allowed as user2 owns repository
			//  and org3 organization that owner user2 is also now has forked this repository
			req := NewRequest(t, "GET", "/user2/repo1")
			resp := session.MakeRequest(t, req, http.StatusOK)
			htmlDoc := NewHTMLParser(t, resp.Body)
			_, exists := htmlDoc.doc.Find("a.ui.button[href^=\"/fork\"]").Attr("href")
			assert.False(t, exists, "Forking should not be allowed anymore")
		})

		t.Run("legacy redirect", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			testRepoForkLegacyRedirect(t, session, "user2", "repo1")
		})
	})
}

func TestForkListPrivateRepo(t *testing.T) {
	forkItemSelector := ".tw-flex.tw-items-center.tw-py-2"

	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		session := loginUser(t, "user5")
		org23 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 23, Visibility: structs.VisibleTypePrivate})

		testRepoFork(t, session, "user2", "repo1", org23.Name, "repo1")

		t.Run("Anonymous", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequest(t, "GET", "/user2/repo1/forks")
			resp := MakeRequest(t, req, http.StatusOK)
			htmlDoc := NewHTMLParser(t, resp.Body)

			htmlDoc.AssertElement(t, forkItemSelector, false)
		})

		t.Run("Logged in", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequest(t, "GET", "/user2/repo1/forks")
			resp := session.MakeRequest(t, req, http.StatusOK)
			htmlDoc := NewHTMLParser(t, resp.Body)

			htmlDoc.AssertElement(t, forkItemSelector, true)
		})
	})
}
