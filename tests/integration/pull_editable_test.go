// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"net/http"
	"net/url"
	"testing"

	auth_model "forgejo.org/models/auth"
	api "forgejo.org/modules/structs"
	"forgejo.org/tests"
)

func TestPullEditable_ShowEditableLabel(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, forgejoURL *url.URL) {
		t.Run("Show editable label if PR is editable", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			editable := true

			setPREditable(t, editable)
			testEditableLabelShown(t, editable)
		})

		t.Run("Don't show editable label if PR is not editable", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			editable := false

			setPREditable(t, editable)
			testEditableLabelShown(t, editable)
		})
	})
}

func setPREditable(t *testing.T, editable bool) {
	t.Helper()
	session := loginUser(t, "user1")
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository)

	req := NewRequestWithJSON(t, "PATCH", "/api/v1/repos/user2/repo1/pulls/3", &api.EditPullRequestOption{
		AllowMaintainerEdit: &editable,
	}).AddTokenAuth(token)
	session.MakeRequest(t, req, http.StatusCreated)
}

func testEditableLabelShown(t *testing.T, expectLabel bool) {
	t.Helper()
	session := loginUser(t, "user2")
	req := NewRequest(t, "GET", "/user2/repo1/pulls/3")
	resp := session.MakeRequest(t, req, http.StatusOK)
	htmlDoc := NewHTMLParser(t, resp.Body)
	htmlDoc.AssertElement(t, "#editable-label", expectLabel)
}
