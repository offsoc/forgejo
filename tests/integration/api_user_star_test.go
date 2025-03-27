// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"testing"

	auth_model "forgejo.org/models/auth"
	"forgejo.org/modules/setting"
	api "forgejo.org/modules/structs"
	"forgejo.org/modules/test"
	"forgejo.org/routers"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
)

func TestAPIStar(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	user := "user1"
	repo := "user2/repo1"

	session := loginUser(t, user)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeReadUser)
	tokenWithUserScope := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteUser, auth_model.AccessTokenScopeWriteRepository)

	assertDisabledStarsNotFound := func(t *testing.T, req *RequestWrapper) {
		t.Helper()

		defer tests.PrintCurrentTest(t)()
		defer test.MockVariableValue(&setting.Repository.DisableStars, true)()
		defer test.MockVariableValue(&testWebRoutes, routers.NormalRoutes())()

		MakeRequest(t, req, http.StatusNotFound)
	}

	t.Run("Star", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, "PUT", fmt.Sprintf("/api/v1/user/starred/%s", repo)).
			AddTokenAuth(tokenWithUserScope)
		MakeRequest(t, req, http.StatusNoContent)

		t.Run("disabled stars", func(t *testing.T) {
			assertDisabledStarsNotFound(t, req)
		})
	})

	t.Run("GetStarredRepos", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, "GET", fmt.Sprintf("/api/v1/users/%s/starred", user)).
			AddTokenAuth(token)
		resp := MakeRequest(t, req, http.StatusOK)

		assert.Equal(t, "1", resp.Header().Get("X-Total-Count"))

		var repos []api.Repository
		DecodeJSON(t, resp, &repos)
		assert.Len(t, repos, 1)
		assert.Equal(t, repo, repos[0].FullName)

		t.Run("disabled stars", func(t *testing.T) {
			assertDisabledStarsNotFound(t, req)
		})
	})

	t.Run("GetMyStarredRepos", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, "GET", "/api/v1/user/starred").
			AddTokenAuth(tokenWithUserScope)
		resp := MakeRequest(t, req, http.StatusOK)

		assert.Equal(t, "1", resp.Header().Get("X-Total-Count"))

		var repos []api.Repository
		DecodeJSON(t, resp, &repos)
		assert.Len(t, repos, 1)
		assert.Equal(t, repo, repos[0].FullName)

		t.Run("disabled stars", func(t *testing.T) {
			assertDisabledStarsNotFound(t, req)
		})
	})

	t.Run("IsStarring", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, "GET", fmt.Sprintf("/api/v1/user/starred/%s", repo)).
			AddTokenAuth(tokenWithUserScope)
		MakeRequest(t, req, http.StatusNoContent)

		req = NewRequest(t, "GET", fmt.Sprintf("/api/v1/user/starred/%s", repo+"notexisting")).
			AddTokenAuth(tokenWithUserScope)
		MakeRequest(t, req, http.StatusNotFound)

		t.Run("disabled stars", func(t *testing.T) {
			assertDisabledStarsNotFound(t, req)
		})
	})

	t.Run("Unstar", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, "DELETE", fmt.Sprintf("/api/v1/user/starred/%s", repo)).
			AddTokenAuth(tokenWithUserScope)
		MakeRequest(t, req, http.StatusNoContent)

		t.Run("disabled stars", func(t *testing.T) {
			assertDisabledStarsNotFound(t, req)
		})
	})
}
