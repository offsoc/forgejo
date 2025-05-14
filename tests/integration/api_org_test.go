// Copyright 2018 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	auth_model "forgejo.org/models/auth"
	"forgejo.org/models/db"
	org_model "forgejo.org/models/organization"
	"forgejo.org/models/perm"
	unit_model "forgejo.org/models/unit"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/setting"
	api "forgejo.org/modules/structs"
	"forgejo.org/modules/test"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
)

func TestAPIOrgCreate(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	token := getUserToken(t, "user1", auth_model.AccessTokenScopeWriteOrganization)

	org := api.CreateOrgOption{
		UserName:    "user1_org",
		FullName:    "User1's organization",
		Description: "This organization created by user1",
		Website:     "https://try.gitea.io",
		Location:    "Shanghai",
		Visibility:  "limited",
	}
	req := NewRequestWithJSON(t, "POST", "/api/v1/orgs", &org).
		AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusCreated)

	var apiOrg api.Organization
	DecodeJSON(t, resp, &apiOrg)

	assert.Equal(t, org.UserName, apiOrg.Name)
	assert.Equal(t, org.FullName, apiOrg.FullName)
	assert.Equal(t, org.Description, apiOrg.Description)
	assert.Equal(t, org.Website, apiOrg.Website)
	assert.Equal(t, org.Location, apiOrg.Location)
	assert.Equal(t, org.Visibility, apiOrg.Visibility)

	unittest.AssertExistsAndLoadBean(t, &user_model.User{
		Name:      org.UserName,
		LowerName: strings.ToLower(org.UserName),
		FullName:  org.FullName,
	})

	// Check owner team permission
	ownerTeam, _ := org_model.GetOwnerTeam(db.DefaultContext, apiOrg.ID)

	for _, ut := range unit_model.AllRepoUnitTypes {
		up := perm.AccessModeOwner
		if ut == unit_model.TypeExternalTracker || ut == unit_model.TypeExternalWiki {
			up = perm.AccessModeRead
		}
		unittest.AssertExistsAndLoadBean(t, &org_model.TeamUnit{
			OrgID:      apiOrg.ID,
			TeamID:     ownerTeam.ID,
			Type:       ut,
			AccessMode: up,
		})
	}

	req = NewRequestf(t, "GET", "/api/v1/orgs/%s", org.UserName).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)
	DecodeJSON(t, resp, &apiOrg)
	assert.Equal(t, org.UserName, apiOrg.Name)

	req = NewRequestf(t, "GET", "/api/v1/orgs/%s/repos", org.UserName).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)

	var repos []*api.Repository
	DecodeJSON(t, resp, &repos)
	for _, repo := range repos {
		assert.False(t, repo.Private)
	}

	req = NewRequestf(t, "GET", "/api/v1/orgs/%s/members", org.UserName).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)

	// user1 on this org is public
	var users []*api.User
	DecodeJSON(t, resp, &users)
	assert.Len(t, users, 1)
	assert.Equal(t, "user1", users[0].UserName)
}

func TestAPIOrgRename(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	token := getUserToken(t, "user1", auth_model.AccessTokenScopeWriteOrganization)

	org := api.CreateOrgOption{
		UserName:    "user1_org",
		FullName:    "User1's organization",
		Description: "This organization created by user1",
		Website:     "https://try.gitea.io",
		Location:    "Shanghai",
		Visibility:  "limited",
	}
	req := NewRequestWithJSON(t, "POST", "/api/v1/orgs", &org).
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusCreated)

	req = NewRequestWithJSON(t, "POST", "/api/v1/orgs/user1_org/rename", &api.RenameOrgOption{
		NewName: "renamed_org",
	}).AddTokenAuth(token)
	MakeRequest(t, req, http.StatusNoContent)
	unittest.AssertExistsAndLoadBean(t, &org_model.Organization{Name: "renamed_org"})
}

func TestAPIOrgEdit(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	session := loginUser(t, "user1")

	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteOrganization)
	org := api.EditOrgOption{
		FullName:    "Org3 organization new full name",
		Description: "A new description",
		Website:     "https://try.gitea.io/new",
		Location:    "Beijing",
		Visibility:  "private",
	}
	req := NewRequestWithJSON(t, "PATCH", "/api/v1/orgs/org3", &org).
		AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusOK)

	var apiOrg api.Organization
	DecodeJSON(t, resp, &apiOrg)

	assert.Equal(t, "org3", apiOrg.Name)
	assert.Equal(t, org.FullName, apiOrg.FullName)
	assert.Equal(t, org.Description, apiOrg.Description)
	assert.Equal(t, org.Website, apiOrg.Website)
	assert.Equal(t, org.Location, apiOrg.Location)
	assert.Equal(t, org.Visibility, apiOrg.Visibility)
}

func TestAPIOrgEditBadVisibility(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	session := loginUser(t, "user1")

	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteOrganization)
	org := api.EditOrgOption{
		FullName:    "Org3 organization new full name",
		Description: "A new description",
		Website:     "https://try.gitea.io/new",
		Location:    "Beijing",
		Visibility:  "badvisibility",
	}
	req := NewRequestWithJSON(t, "PATCH", "/api/v1/orgs/org3", &org).
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusUnprocessableEntity)
}

func TestAPIOrgDeny(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	defer test.MockVariableValue(&setting.Service.RequireSignInView, true)()

	orgName := "user1_org"
	req := NewRequestf(t, "GET", "/api/v1/orgs/%s", orgName)
	MakeRequest(t, req, http.StatusNotFound)

	req = NewRequestf(t, "GET", "/api/v1/orgs/%s/repos", orgName)
	MakeRequest(t, req, http.StatusNotFound)

	req = NewRequestf(t, "GET", "/api/v1/orgs/%s/members", orgName)
	MakeRequest(t, req, http.StatusNotFound)
}

func TestAPIGetAll(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	token := getUserToken(t, "user1", auth_model.AccessTokenScopeReadPackage)

	// accessing with a token will return all orgs
	req := NewRequest(t, "GET", "/api/v1/orgs").
		AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusOK)
	var apiOrgList []*api.Organization

	DecodeJSON(t, resp, &apiOrgList)
	assert.Len(t, apiOrgList, 12)
	assert.Equal(t, "Limited Org 36", apiOrgList[1].FullName)
	assert.Equal(t, "limited", apiOrgList[1].Visibility)

	// accessing without a token will return only public orgs
	req = NewRequest(t, "GET", "/api/v1/orgs")
	resp = MakeRequest(t, req, http.StatusOK)

	DecodeJSON(t, resp, &apiOrgList)
	assert.Len(t, apiOrgList, 8)
	assert.Equal(t, "org 17", apiOrgList[0].FullName)
	assert.Equal(t, "public", apiOrgList[0].Visibility)
}

func TestAPIOrgSearchEmptyTeam(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	token := getUserToken(t, "user1", auth_model.AccessTokenScopeWriteOrganization)
	orgName := "org_with_empty_team"

	// create org
	req := NewRequestWithJSON(t, "POST", "/api/v1/orgs", &api.CreateOrgOption{
		UserName: orgName,
	}).AddTokenAuth(token)
	MakeRequest(t, req, http.StatusCreated)

	// create team with no member
	req = NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/orgs/%s/teams", orgName), &api.CreateTeamOption{
		Name:                    "Empty",
		IncludesAllRepositories: true,
		Permission:              "read",
		Units:                   []string{"repo.code", "repo.issues", "repo.ext_issues", "repo.wiki", "repo.pulls"},
	}).AddTokenAuth(token)
	MakeRequest(t, req, http.StatusCreated)

	// case-insensitive search for teams that have no members
	req = NewRequest(t, "GET", fmt.Sprintf("/api/v1/orgs/%s/teams/search?q=%s", orgName, "empty")).
		AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusOK)
	data := struct {
		Ok   bool
		Data []*api.Team
	}{}
	DecodeJSON(t, resp, &data)
	assert.True(t, data.Ok)
	if assert.Len(t, data.Data, 1) {
		assert.Equal(t, "Empty", data.Data[0].Name)
	}
}

func TestAPIOrgChangeEmail(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	session := loginUser(t, "user1")
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteOrganization)

	t.Run("Invalid", func(t *testing.T) {
		newMail := "invalid"
		settings := api.EditOrgOption{Email: &newMail}

		resp := MakeRequest(t, NewRequestWithJSON(t, "PATCH", "/api/v1/orgs/org3", &settings).AddTokenAuth(token), http.StatusUnprocessableEntity)

		var org *api.Organization
		DecodeJSON(t, resp, &org)

		assert.Empty(t, org.Email)
	})

	t.Run("Valid", func(t *testing.T) {
		newMail := "example@example.com"
		settings := api.EditOrgOption{Email: &newMail}

		resp := MakeRequest(t, NewRequestWithJSON(t, "PATCH", "/api/v1/orgs/org3", &settings).AddTokenAuth(token), http.StatusOK)

		var org *api.Organization
		DecodeJSON(t, resp, &org)

		assert.Equal(t, "example@example.com", org.Email)
	})

	t.Run("NoChange", func(t *testing.T) {
		settings := api.EditOrgOption{}

		resp := MakeRequest(t, NewRequestWithJSON(t, "PATCH", "/api/v1/orgs/org3", &settings).AddTokenAuth(token), http.StatusOK)

		var org *api.Organization
		DecodeJSON(t, resp, &org)

		assert.Equal(t, "example@example.com", org.Email)
	})

	t.Run("Empty", func(t *testing.T) {
		newMail := ""
		settings := api.EditOrgOption{Email: &newMail}

		resp := MakeRequest(t, NewRequestWithJSON(t, "PATCH", "/api/v1/orgs/org3", &settings).AddTokenAuth(token), http.StatusOK)

		var org *api.Organization
		DecodeJSON(t, resp, &org)

		assert.Empty(t, org.Email)
	})
}

func TestPublicOrganizationsAccess(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	tokens := map[string]string{
		"user1":     getTokenForLoggedInUser(t, loginUser(t, "user1"), auth_model.AccessTokenScopeReadPackage),
		"user2":     getTokenForLoggedInUser(t, loginUser(t, "user2"), auth_model.AccessTokenScopeReadPackage),
		"user5":     getTokenForLoggedInUser(t, loginUser(t, "user5"), auth_model.AccessTokenScopeReadPackage),
		"anonymous": "",
	}

	tests := []struct {
		tokenName string
		orga      string
		status    int
	}{
		{"user2", "", http.StatusOK},
		{"user5", "", http.StatusOK},
		{"anonymous", "", http.StatusOK},
		{"user2", "privated_org", http.StatusNotFound},
		{"user5", "privated_org", http.StatusOK},
		{"anonymous", "privated_org", http.StatusNotFound},
		{"user1", "private_org35", http.StatusOK},
		{"user5", "private_org35", http.StatusNotFound},
		{"anonymous", "private_org35", http.StatusNotFound},
	}

	for _, tt := range tests {
		url := strings.TrimSuffix(fmt.Sprintf("/api/v1/orgs/%s", tt.orga), "/")
		name := fmt.Sprintf("%s, %s %s", url, tt.tokenName, func() string {
			if tt.status == http.StatusOK {
				return "should see the orga"
			}

			return "should not see orga"
		}())

		t.Run(name, func(t *testing.T) {
			req := NewRequest(t, "GET", url)

			if token, ok := tokens[tt.tokenName]; ok && token != "" {
				req.AddTokenAuth(token)
			}

			MakeRequest(t, req, tt.status)
		})
	}
}
