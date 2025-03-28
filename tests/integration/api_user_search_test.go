// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"testing"

	auth_model "forgejo.org/models/auth"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/setting"
	api "forgejo.org/modules/structs"
	"forgejo.org/modules/test"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
)

type SearchResults struct {
	OK   bool        `json:"ok"`
	Data []*api.User `json:"data"`
}

func TestAPIUserSearchLoggedIn(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	adminUsername := "user1"
	session := loginUser(t, adminUsername)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeReadUser)
	query := "user2"
	req := NewRequestf(t, "GET", "/api/v1/users/search?q=%s", query).
		AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusOK)

	var results SearchResults
	DecodeJSON(t, resp, &results)
	assert.NotEmpty(t, results.Data)
	for _, user := range results.Data {
		assert.Contains(t, user.UserName, query)
		assert.NotEmpty(t, user.Email)
	}

	publicToken := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeReadUser, auth_model.AccessTokenScopePublicOnly)
	req = NewRequestf(t, "GET", "/api/v1/users/search?q=%s", query).
		AddTokenAuth(publicToken)
	resp = MakeRequest(t, req, http.StatusOK)
	results = SearchResults{}
	DecodeJSON(t, resp, &results)
	assert.NotEmpty(t, results.Data)
	for _, user := range results.Data {
		assert.Contains(t, user.UserName, query)
		assert.NotEmpty(t, user.Email)
		assert.Equal(t, "public", user.Visibility)
	}
}

func TestAPIUserSearchNotLoggedIn(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	query := "user2"
	req := NewRequestf(t, "GET", "/api/v1/users/search?q=%s", query)
	resp := MakeRequest(t, req, http.StatusOK)

	var results SearchResults
	DecodeJSON(t, resp, &results)
	assert.NotEmpty(t, results.Data)
	var modelUser *user_model.User
	for _, user := range results.Data {
		assert.Contains(t, user.UserName, query)
		modelUser = unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: user.ID})
		assert.Equal(t, modelUser.GetPlaceholderEmail(), user.Email)
	}
}

func TestAPIUserSearchPaged(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	defer test.MockVariableValue(&setting.API.DefaultPagingNum, 5)()

	req := NewRequest(t, "GET", "/api/v1/users/search?limit=1")
	resp := MakeRequest(t, req, http.StatusOK)

	var limitedResults SearchResults
	DecodeJSON(t, resp, &limitedResults)
	assert.Len(t, limitedResults.Data, 1)

	req = NewRequest(t, "GET", "/api/v1/users/search")
	resp = MakeRequest(t, req, http.StatusOK)

	var results SearchResults
	DecodeJSON(t, resp, &results)
	assert.Len(t, results.Data, 5)
}

func TestAPIUserSearchSystemUsers(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	for _, systemUser := range []*user_model.User{
		user_model.NewGhostUser(),
		user_model.NewActionsUser(),
	} {
		t.Run(systemUser.Name, func(t *testing.T) {
			req := NewRequestf(t, "GET", "/api/v1/users/search?uid=%d", systemUser.ID)
			resp := MakeRequest(t, req, http.StatusOK)

			var results SearchResults
			DecodeJSON(t, resp, &results)
			assert.NotEmpty(t, results.Data)
			if assert.Len(t, results.Data, 1) {
				user := results.Data[0]
				assert.Equal(t, user.UserName, systemUser.Name)
				assert.Equal(t, user.ID, systemUser.ID)
			}
		})
	}
}

func TestAPIUserSearchAdminLoggedInUserHidden(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	adminUsername := "user1"
	session := loginUser(t, adminUsername)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeReadUser)
	query := "user31"
	req := NewRequestf(t, "GET", "/api/v1/users/search?q=%s", query).
		AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusOK)

	var results SearchResults
	DecodeJSON(t, resp, &results)
	assert.NotEmpty(t, results.Data)
	for _, user := range results.Data {
		assert.Contains(t, user.UserName, query)
		assert.NotEmpty(t, user.Email)
		assert.Equal(t, "private", user.Visibility)
	}
}

func TestAPIUserSearchNotLoggedInUserHidden(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	query := "user31"
	req := NewRequestf(t, "GET", "/api/v1/users/search?q=%s", query)
	resp := MakeRequest(t, req, http.StatusOK)

	var results SearchResults
	DecodeJSON(t, resp, &results)
	assert.Empty(t, results.Data)
}

func TestAPIUserSearchByEmail(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// admin can search user with private email
	adminUsername := "user1"
	session := loginUser(t, adminUsername)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeReadUser)
	query := "user2@example.com"
	req := NewRequestf(t, "GET", "/api/v1/users/search?q=%s", query).
		AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusOK)

	var results SearchResults
	DecodeJSON(t, resp, &results)
	assert.Len(t, results.Data, 1)
	assert.Equal(t, query, results.Data[0].Email)

	// no login user can not search user with private email
	req = NewRequestf(t, "GET", "/api/v1/users/search?q=%s", query)
	resp = MakeRequest(t, req, http.StatusOK)
	DecodeJSON(t, resp, &results)
	assert.Empty(t, results.Data)

	// user can search self with private email
	user2 := "user2"
	session = loginUser(t, user2)
	token = getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeReadUser)
	req = NewRequestf(t, "GET", "/api/v1/users/search?q=%s", query).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)

	DecodeJSON(t, resp, &results)
	assert.Len(t, results.Data, 1)
	assert.Equal(t, query, results.Data[0].Email)
}
