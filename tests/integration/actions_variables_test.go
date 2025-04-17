// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"fmt"
	"net/http"
	"testing"

	actions_model "forgejo.org/models/actions"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	forgejo_context "forgejo.org/services/context"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
)

func TestActionVariablesModification(t *testing.T) {
	defer tests.AddFixtures("tests/integration/fixtures/TestActionVariablesModification")()
	defer tests.PrepareTestEnv(t)()

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	userVariable := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionVariable{ID: 1001, OwnerID: user.ID})
	userURL := "/user/settings/actions/variables"
	org := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 3, Type: user_model.UserTypeOrganization})
	orgVariable := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionVariable{ID: 1002, OwnerID: org.ID})
	orgURL := "/org/" + org.Name + "/settings/actions/variables"
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1, OwnerID: user.ID})
	repoVariable := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionVariable{ID: 1003, RepoID: repo.ID})
	repoURL := "/" + repo.FullName() + "/settings/actions/variables"
	admin := unittest.AssertExistsAndLoadBean(t, &user_model.User{IsAdmin: true})
	globalVariable := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionVariable{ID: 1004}, "owner_id = 0 AND repo_id = 0")
	adminURL := "/admin/actions/variables"

	adminSess := loginUser(t, admin.Name)
	adminCSRF := GetCSRF(t, adminSess, "/")
	sess := loginUser(t, user.Name)
	csrf := GetCSRF(t, sess, "/")

	type errorJSON struct {
		Error string `json:"errorMessage"`
	}

	test := func(t *testing.T, fail bool, baseURL string, id int64) {
		defer tests.PrintCurrentTest(t, 1)()
		t.Helper()

		sess := sess
		csrf := csrf
		if baseURL == adminURL {
			sess = adminSess
			csrf = adminCSRF
		}

		req := NewRequestWithValues(t, "POST", baseURL+fmt.Sprintf("/%d/edit", id), map[string]string{
			"_csrf": csrf,
			"name":  "glados_quote",
			"data":  "I'm fine. Two plus two is...ten, in base four, I'm fine!",
		})
		if fail {
			resp := sess.MakeRequest(t, req, http.StatusBadRequest)
			var error errorJSON
			DecodeJSON(t, resp, &error)
			assert.Equal(t, "Failed to find the variable.", error.Error)
		} else {
			sess.MakeRequest(t, req, http.StatusOK)
			flashCookie := sess.GetCookie(forgejo_context.CookieNameFlash)
			assert.NotNil(t, flashCookie)
			assert.Equal(t, "success%3DThe%2Bvariable%2Bhas%2Bbeen%2Bedited.", flashCookie.Value)
		}

		req = NewRequestWithValues(t, "POST", baseURL+fmt.Sprintf("/%d/delete", id), map[string]string{
			"_csrf": csrf,
		})
		if fail {
			resp := sess.MakeRequest(t, req, http.StatusBadRequest)
			var error errorJSON
			DecodeJSON(t, resp, &error)
			assert.Equal(t, "Failed to find the variable.", error.Error)
		} else {
			sess.MakeRequest(t, req, http.StatusOK)
			flashCookie := sess.GetCookie(forgejo_context.CookieNameFlash)
			assert.NotNil(t, flashCookie)
			assert.Equal(t, "success%3DThe%2Bvariable%2Bhas%2Bbeen%2Bremoved.", flashCookie.Value)
		}
	}

	t.Run("User variable", func(t *testing.T) {
		t.Run("Organisation", func(t *testing.T) {
			test(t, true, orgURL, userVariable.ID)
		})
		t.Run("Repository", func(t *testing.T) {
			test(t, true, repoURL, userVariable.ID)
		})
		t.Run("Admin", func(t *testing.T) {
			test(t, true, adminURL, userVariable.ID)
		})
		t.Run("User", func(t *testing.T) {
			test(t, false, userURL, userVariable.ID)
		})
	})

	t.Run("Organisation variable", func(t *testing.T) {
		t.Run("Repository", func(t *testing.T) {
			test(t, true, repoURL, orgVariable.ID)
		})
		t.Run("User", func(t *testing.T) {
			test(t, true, userURL, orgVariable.ID)
		})
		t.Run("Admin", func(t *testing.T) {
			test(t, true, adminURL, userVariable.ID)
		})
		t.Run("Organisation", func(t *testing.T) {
			test(t, false, orgURL, orgVariable.ID)
		})
	})

	t.Run("Repository variable", func(t *testing.T) {
		t.Run("Organisation", func(t *testing.T) {
			test(t, true, orgURL, repoVariable.ID)
		})
		t.Run("User", func(t *testing.T) {
			test(t, true, userURL, repoVariable.ID)
		})
		t.Run("Admin", func(t *testing.T) {
			test(t, true, adminURL, userVariable.ID)
		})
		t.Run("Repository", func(t *testing.T) {
			test(t, false, repoURL, repoVariable.ID)
		})
	})

	t.Run("Global variable", func(t *testing.T) {
		t.Run("Organisation", func(t *testing.T) {
			test(t, true, orgURL, globalVariable.ID)
		})
		t.Run("User", func(t *testing.T) {
			test(t, true, userURL, globalVariable.ID)
		})
		t.Run("Repository", func(t *testing.T) {
			test(t, true, repoURL, globalVariable.ID)
		})
		t.Run("Admin", func(t *testing.T) {
			test(t, false, adminURL, globalVariable.ID)
		})
	})
}
