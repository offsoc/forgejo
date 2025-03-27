// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	git_model "forgejo.org/models/git"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	gitea_context "forgejo.org/services/context"

	"github.com/stretchr/testify/assert"
)

func TestBranchActions(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		session := loginUser(t, "user2")
		repo1 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
		branch3 := unittest.AssertExistsAndLoadBean(t, &git_model.Branch{ID: 3, RepoID: repo1.ID})
		branchesLink := repo1.FullName() + "/branches"

		t.Run("View", func(t *testing.T) {
			req := NewRequest(t, "GET", branchesLink)
			MakeRequest(t, req, http.StatusOK)
		})

		t.Run("Delete branch", func(t *testing.T) {
			link := fmt.Sprintf("/%s/branches/delete?name=%s", repo1.FullName(), branch3.Name)
			req := NewRequestWithValues(t, "POST", link, map[string]string{
				"_csrf": GetCSRF(t, session, branchesLink),
			})
			session.MakeRequest(t, req, http.StatusOK)
			flashCookie := session.GetCookie(gitea_context.CookieNameFlash)
			assert.NotNil(t, flashCookie)
			assert.Contains(t, flashCookie.Value, "success%3DBranch%2B%2522branch2%2522%2Bhas%2Bbeen%2Bdeleted.")

			assert.True(t, unittest.AssertExistsAndLoadBean(t, &git_model.Branch{ID: 3, RepoID: repo1.ID}).IsDeleted)
		})

		t.Run("Restore branch", func(t *testing.T) {
			link := fmt.Sprintf("/%s/branches/restore?branch_id=%d&name=%s", repo1.FullName(), branch3.ID, branch3.Name)
			req := NewRequestWithValues(t, "POST", link, map[string]string{
				"_csrf": GetCSRF(t, session, branchesLink),
			})
			session.MakeRequest(t, req, http.StatusOK)
			flashCookie := session.GetCookie(gitea_context.CookieNameFlash)
			assert.NotNil(t, flashCookie)
			assert.Contains(t, flashCookie.Value, "success%3DBranch%2B%2522branch2%2522%2Bhas%2Bbeen%2Brestored")

			assert.False(t, unittest.AssertExistsAndLoadBean(t, &git_model.Branch{ID: 3, RepoID: repo1.ID}).IsDeleted)
		})
	})
}
