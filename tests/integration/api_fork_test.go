// Copyright 2017 The Gogs Authors. All rights reserved.
// Copyright 2024 The Forgejo Authors c/o Codeberg e.V.. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"testing"

	auth_model "forgejo.org/models/auth"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/setting"
	api "forgejo.org/modules/structs"
	"forgejo.org/modules/test"
	"forgejo.org/routers"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
)

func TestAPIForkAsAdminIgnoringLimits(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	defer test.MockVariableValue(&setting.Repository.AllowForkWithoutMaximumLimit, false)()
	defer test.MockVariableValue(&setting.Repository.MaxCreationLimit, 0)()

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: "user2"})
	userSession := loginUser(t, user.Name)
	userToken := getTokenForLoggedInUser(t, userSession, auth_model.AccessTokenScopeWriteRepository)
	adminUser := unittest.AssertExistsAndLoadBean(t, &user_model.User{IsAdmin: true})
	adminSession := loginUser(t, adminUser.Name)
	adminToken := getTokenForLoggedInUser(t, adminSession,
		auth_model.AccessTokenScopeWriteRepository,
		auth_model.AccessTokenScopeWriteOrganization)

	originForkURL := "/api/v1/repos/user12/repo10/forks"
	orgName := "fork-org"

	// Create an organization
	req := NewRequestWithJSON(t, "POST", "/api/v1/orgs", &api.CreateOrgOption{
		UserName: orgName,
	}).AddTokenAuth(adminToken)
	MakeRequest(t, req, http.StatusCreated)

	// Create a team
	teamToCreate := &api.CreateTeamOption{
		Name:                    "testers",
		IncludesAllRepositories: true,
		Permission:              "write",
		Units:                   []string{"repo.code", "repo.issues"},
	}

	req = NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/orgs/%s/teams", orgName), &teamToCreate).AddTokenAuth(adminToken)
	resp := MakeRequest(t, req, http.StatusCreated)
	var team api.Team
	DecodeJSON(t, resp, &team)

	// Add user2 to the team
	req = NewRequestf(t, "PUT", "/api/v1/teams/%d/members/user2", team.ID).AddTokenAuth(adminToken)
	MakeRequest(t, req, http.StatusNoContent)

	t.Run("forking as regular user", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequestWithJSON(t, "POST", originForkURL, &api.CreateForkOption{
			Organization: &orgName,
		}).AddTokenAuth(userToken)
		MakeRequest(t, req, http.StatusConflict)
	})

	t.Run("forking as an instance admin", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequestWithJSON(t, "POST", originForkURL, &api.CreateForkOption{
			Organization: &orgName,
		}).AddTokenAuth(adminToken)
		MakeRequest(t, req, http.StatusAccepted)
	})
}

func TestCreateForkNoLogin(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	req := NewRequestWithJSON(t, "POST", "/api/v1/repos/user2/repo1/forks", &api.CreateForkOption{})
	MakeRequest(t, req, http.StatusUnauthorized)
}

func TestAPIDisabledForkRepo(t *testing.T) {
	defer test.MockVariableValue(&setting.Repository.DisableForks, true)()
	defer test.MockVariableValue(&testWebRoutes, routers.NormalRoutes())()
	defer tests.PrepareTestEnv(t)()

	t.Run("fork listing", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, "GET", "/api/v1/repos/user2/repo1/forks")
		MakeRequest(t, req, http.StatusNotFound)
	})

	t.Run("forking", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		session := loginUser(t, "user5")
		token := getTokenForLoggedInUser(t, session)

		req := NewRequestWithJSON(t, "POST", "/api/v1/repos/user2/repo1/forks", &api.CreateForkOption{}).AddTokenAuth(token)
		session.MakeRequest(t, req, http.StatusNotFound)
	})
}

func TestAPIForkListPrivateRepo(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	session := loginUser(t, "user5")
	token := getTokenForLoggedInUser(t, session,
		auth_model.AccessTokenScopeWriteRepository,
		auth_model.AccessTokenScopeWriteOrganization)
	org23 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 23, Visibility: api.VisibleTypePrivate})

	req := NewRequestWithJSON(t, "POST", "/api/v1/repos/user2/repo1/forks", &api.CreateForkOption{
		Organization: &org23.Name,
	}).AddTokenAuth(token)
	MakeRequest(t, req, http.StatusAccepted)

	t.Run("Anonymous", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, "GET", "/api/v1/repos/user2/repo1/forks")
		resp := MakeRequest(t, req, http.StatusOK)

		var forks []*api.Repository
		DecodeJSON(t, resp, &forks)

		assert.Empty(t, forks)
		assert.Equal(t, "0", resp.Header().Get("X-Total-Count"))
	})

	t.Run("Logged in", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, "GET", "/api/v1/repos/user2/repo1/forks").AddTokenAuth(token)
		resp := MakeRequest(t, req, http.StatusOK)

		var forks []*api.Repository
		DecodeJSON(t, resp, &forks)

		assert.Len(t, forks, 1)
		assert.Equal(t, "1", resp.Header().Get("X-Total-Count"))
	})
}
