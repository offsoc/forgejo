// Copyright 2024 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"testing"

	auth_model "forgejo.org/models/auth"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	api "forgejo.org/modules/structs"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
)

func TestAPICompareBranches(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	// Login as User2.
	session := loginUser(t, user.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository)

	repoName := "repo20"

	req := NewRequestf(t, "GET", "/api/v1/repos/user2/%s/compare/add-csv...remove-files-b", repoName).
		AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusOK)

	var apiResp *api.Compare
	DecodeJSON(t, resp, &apiResp)

	assert.Equal(t, 2, apiResp.TotalCommits)
	assert.Len(t, apiResp.Commits, 2)
	assert.Len(t, apiResp.Files, 3)
}

func TestAPICompareCommits(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	// Login as User2.
	session := loginUser(t, user.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository)

	req := NewRequestf(t, "GET", "/api/v1/repos/user2/repo20/compare/c8e31bc...8babce9").
		AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusOK)

	var apiResp *api.Compare
	DecodeJSON(t, resp, &apiResp)

	assert.Equal(t, 2, apiResp.TotalCommits)
	assert.Len(t, apiResp.Commits, 2)
	assert.Len(t, apiResp.Files, 3)
}
