// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"testing"

	auth_model "forgejo.org/models/auth"
	api "forgejo.org/modules/structs"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
)

func getOrgSettingsFormData(t *testing.T, session *TestSession, orgName string) map[string]string {
	return map[string]string{
		"_csrf":                         GetCSRF(t, session, fmt.Sprintf("/org/%s/settings", orgName)),
		"name":                          orgName,
		"full_name":                     "",
		"email":                         "",
		"description":                   "",
		"website":                       "",
		"location":                      "",
		"visibility":                    "0",
		"repo_admin_change_team_access": "on",
		"max_repo_creation":             "-1",
	}
}

func getOrgSettings(t *testing.T, token, orgName string) *api.Organization {
	t.Helper()

	req := NewRequestf(t, "GET", "/api/v1/orgs/%s", orgName).AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusOK)

	var org *api.Organization
	DecodeJSON(t, resp, &org)

	return org
}

func TestOrgSettingsChangeEmail(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	const orgName = "org3"
	settingsURL := fmt.Sprintf("/org/%s/settings", orgName)

	session := loginUser(t, "user1")
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeReadOrganization)

	t.Run("Invalid", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		settings := getOrgSettingsFormData(t, session, orgName)

		settings["email"] = "invalid"
		session.MakeRequest(t, NewRequestWithValues(t, "POST", settingsURL, settings), http.StatusOK)

		org := getOrgSettings(t, token, orgName)
		assert.Equal(t, "org3@example.com", org.Email)
	})

	t.Run("Valid", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		settings := getOrgSettingsFormData(t, session, orgName)

		settings["email"] = "example@example.com"
		session.MakeRequest(t, NewRequestWithValues(t, "POST", settingsURL, settings), http.StatusSeeOther)

		org := getOrgSettings(t, token, orgName)
		assert.Equal(t, "example@example.com", org.Email)
	})

	t.Run("Empty", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		settings := getOrgSettingsFormData(t, session, orgName)

		settings["email"] = ""
		session.MakeRequest(t, NewRequestWithValues(t, "POST", settingsURL, settings), http.StatusSeeOther)

		org := getOrgSettings(t, token, orgName)
		assert.Empty(t, org.Email)
	})
}
