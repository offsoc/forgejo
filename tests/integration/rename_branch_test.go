// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"net/url"
	"testing"

	git_model "forgejo.org/models/git"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	gitea_context "forgejo.org/services/context"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
)

func TestRenameBranch(t *testing.T) {
	onGiteaRun(t, testRenameBranch)
}

func testRenameBranch(t *testing.T, u *url.URL) {
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	unittest.AssertExistsAndLoadBean(t, &git_model.Branch{RepoID: repo.ID, Name: "master"})

	// get branch setting page
	session := loginUser(t, "user2")
	t.Run("Normal", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequestWithValues(t, "POST", "/user2/repo1/settings/rename_branch", map[string]string{
			"_csrf": GetCSRF(t, session, "/user2/repo1/settings/branches"),
			"from":  "master",
			"to":    "main",
		})
		session.MakeRequest(t, req, http.StatusSeeOther)

		// check new branch link
		req = NewRequest(t, "GET", "/user2/repo1/src/branch/main/README.md")
		session.MakeRequest(t, req, http.StatusOK)

		// check old branch link
		req = NewRequest(t, "GET", "/user2/repo1/src/branch/master/README.md")
		resp := session.MakeRequest(t, req, http.StatusSeeOther)
		location := resp.Header().Get("Location")
		assert.Equal(t, "/user2/repo1/src/branch/main/README.md", location)

		// check db
		repo1 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
		assert.Equal(t, "main", repo1.DefaultBranch)
	})

	t.Run("Database synchronization", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequestWithValues(t, "POST", "/user2/repo1/settings/rename_branch", map[string]string{
			"_csrf": GetCSRF(t, session, "/user2/repo1/settings/branches"),
			"from":  "master",
			"to":    "main",
		})
		session.MakeRequest(t, req, http.StatusSeeOther)

		// check new branch link
		req = NewRequestWithValues(t, "GET", "/user2/repo1/src/branch/main/README.md", nil)
		session.MakeRequest(t, req, http.StatusOK)

		// check old branch link
		req = NewRequestWithValues(t, "GET", "/user2/repo1/src/branch/master/README.md", nil)
		resp := session.MakeRequest(t, req, http.StatusSeeOther)
		location := resp.Header().Get("Location")
		assert.Equal(t, "/user2/repo1/src/branch/main/README.md", location)

		// check db
		repo1 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
		assert.Equal(t, "main", repo1.DefaultBranch)

		// create branch1
		csrf := GetCSRF(t, session, "/user2/repo1/src/branch/main")

		req = NewRequestWithValues(t, "POST", "/user2/repo1/branches/_new/branch/main", map[string]string{
			"_csrf":           csrf,
			"new_branch_name": "branch1",
		})
		session.MakeRequest(t, req, http.StatusSeeOther)

		branch1 := unittest.AssertExistsAndLoadBean(t, &git_model.Branch{RepoID: repo1.ID, Name: "branch1"})
		assert.Equal(t, "branch1", branch1.Name)

		// create branch2
		req = NewRequestWithValues(t, "POST", "/user2/repo1/branches/_new/branch/main", map[string]string{
			"_csrf":           csrf,
			"new_branch_name": "branch2",
		})
		session.MakeRequest(t, req, http.StatusSeeOther)

		branch2 := unittest.AssertExistsAndLoadBean(t, &git_model.Branch{RepoID: repo1.ID, Name: "branch2"})
		assert.Equal(t, "branch2", branch2.Name)

		// rename branch2 to branch1
		req = NewRequestWithValues(t, "POST", "/user2/repo1/settings/rename_branch", map[string]string{
			"_csrf": GetCSRF(t, session, "/user2/repo1/settings/branches"),
			"from":  "branch2",
			"to":    "branch1",
		})
		session.MakeRequest(t, req, http.StatusSeeOther)
		flashCookie := session.GetCookie(gitea_context.CookieNameFlash)
		assert.NotNil(t, flashCookie)
		assert.Contains(t, flashCookie.Value, "error")

		branch2 = unittest.AssertExistsAndLoadBean(t, &git_model.Branch{RepoID: repo1.ID, Name: "branch2"})
		assert.Equal(t, "branch2", branch2.Name)
		branch1 = unittest.AssertExistsAndLoadBean(t, &git_model.Branch{RepoID: repo1.ID, Name: "branch1"})
		assert.Equal(t, "branch1", branch1.Name)

		// delete branch1
		req = NewRequestWithValues(t, "POST", "/user2/repo1/branches/delete", map[string]string{
			"_csrf": GetCSRF(t, session, "/user2/repo1/settings/branches"),
			"name":  "branch1",
		})
		session.MakeRequest(t, req, http.StatusOK)
		branch2 = unittest.AssertExistsAndLoadBean(t, &git_model.Branch{RepoID: repo1.ID, Name: "branch2"})
		assert.Equal(t, "branch2", branch2.Name)
		branch1 = unittest.AssertExistsAndLoadBean(t, &git_model.Branch{RepoID: repo1.ID, Name: "branch1"})
		assert.True(t, branch1.IsDeleted) // virtual deletion

		// rename branch2 to branch1 again
		req = NewRequestWithValues(t, "POST", "/user2/repo1/settings/rename_branch", map[string]string{
			"_csrf": GetCSRF(t, session, "/user2/repo1/settings/branches"),
			"from":  "branch2",
			"to":    "branch1",
		})
		session.MakeRequest(t, req, http.StatusSeeOther)

		flashCookie = session.GetCookie(gitea_context.CookieNameFlash)
		assert.NotNil(t, flashCookie)
		assert.Contains(t, flashCookie.Value, "success")

		unittest.AssertNotExistsBean(t, &git_model.Branch{RepoID: repo1.ID, Name: "branch2"})
		branch1 = unittest.AssertExistsAndLoadBean(t, &git_model.Branch{RepoID: repo1.ID, Name: "branch1"})
		assert.Equal(t, "branch1", branch1.Name)
	})

	t.Run("Protected branch", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		// Add protected branch
		req := NewRequestWithValues(t, "POST", "/user2/repo1/settings/branches/edit", map[string]string{
			"_csrf":       GetCSRF(t, session, "/user2/repo1/settings/branches/edit"),
			"rule_name":   "*",
			"enable_push": "true",
		})
		session.MakeRequest(t, req, http.StatusSeeOther)

		// Verify it was added.
		unittest.AssertExistsIf(t, true, &git_model.ProtectedBranch{RuleName: "*", RepoID: repo.ID})

		req = NewRequestWithValues(t, "POST", "/user2/repo1/settings/rename_branch", map[string]string{
			"_csrf": GetCSRF(t, session, "/user2/repo1/settings/branches"),
			"from":  "main",
			"to":    "main2",
		})
		session.MakeRequest(t, req, http.StatusSeeOther)

		flashCookie := session.GetCookie(gitea_context.CookieNameFlash)
		assert.NotNil(t, flashCookie)
		assert.Equal(t, "error%3DCannot%2Brename%2Bbranch%2Bmain2%2Bbecause%2Bit%2Bis%2Ba%2Bprotected%2Bbranch.", flashCookie.Value)

		// Verify it didn't change.
		repo1 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
		assert.Equal(t, "main", repo1.DefaultBranch)
	})
}
