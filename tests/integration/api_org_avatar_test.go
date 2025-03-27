// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"encoding/base64"
	"net/http"
	"os"
	"testing"

	auth_model "forgejo.org/models/auth"
	api "forgejo.org/modules/structs"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIUpdateOrgAvatar(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	session := loginUser(t, "user1")

	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteOrganization)

	// Test what happens if you use a valid image
	avatar, err := os.ReadFile("tests/integration/avatar.png")
	require.NoError(t, err)
	if err != nil {
		assert.FailNow(t, "Unable to open avatar.png")
	}

	opts := api.UpdateUserAvatarOption{
		Image: base64.StdEncoding.EncodeToString(avatar),
	}

	req := NewRequestWithJSON(t, "POST", "/api/v1/orgs/org3/avatar", &opts).
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusNoContent)

	// Test what happens if you don't have a valid Base64 string
	opts = api.UpdateUserAvatarOption{
		Image: "Invalid",
	}

	req = NewRequestWithJSON(t, "POST", "/api/v1/orgs/org3/avatar", &opts).
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusBadRequest)

	// Test what happens if you use a file that is not an image
	text, err := os.ReadFile("tests/integration/README.md")
	require.NoError(t, err)
	if err != nil {
		assert.FailNow(t, "Unable to open README.md")
	}

	opts = api.UpdateUserAvatarOption{
		Image: base64.StdEncoding.EncodeToString(text),
	}

	req = NewRequestWithJSON(t, "POST", "/api/v1/orgs/org3/avatar", &opts).
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusInternalServerError)
}

func TestAPIDeleteOrgAvatar(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	session := loginUser(t, "user1")

	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteOrganization)

	req := NewRequest(t, "DELETE", "/api/v1/orgs/org3/avatar").
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusNoContent)
}
