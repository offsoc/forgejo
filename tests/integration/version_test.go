// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"testing"

	auth_model "forgejo.org/models/auth"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/structs"
	"forgejo.org/modules/test"
	"forgejo.org/routers"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
)

func TestVersion(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	t.Run("Version", func(t *testing.T) {
		setting.AppVer = "test-version-1"
		req := NewRequest(t, "GET", "/api/v1/version")
		resp := MakeRequest(t, req, http.StatusOK)

		var version structs.ServerVersion
		DecodeJSON(t, resp, &version)
		assert.Equal(t, setting.AppVer, version.Version)
	})

	t.Run("Versions with REQUIRE_SIGNIN_VIEW enabled", func(t *testing.T) {
		defer test.MockVariableValue(&setting.Service.RequireSignInView, true)()
		defer test.MockVariableValue(&testWebRoutes, routers.NormalRoutes())()

		setting.AppVer = "test-version-1"

		t.Run("Get version without auth", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			// GET api without auth
			req := NewRequest(t, "GET", "/api/v1/version")
			MakeRequest(t, req, http.StatusForbidden)
		})

		t.Run("Get version without auth", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			username := "user1"
			session := loginUser(t, username)
			token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository)

			// GET api with auth
			req := NewRequest(t, "GET", "/api/v1/version").AddTokenAuth(token)
			resp := MakeRequest(t, req, http.StatusOK)

			var version structs.ServerVersion
			DecodeJSON(t, resp, &version)
			assert.Equal(t, setting.AppVer, version.Version)
		})
	})
}
