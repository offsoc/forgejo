// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/activitypub"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/test"
	"code.gitea.io/gitea/routers"
	"code.gitea.io/gitea/tests"

	ap "github.com/go-ap/activitypub"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActivityPubPerson(t *testing.T) {
	defer test.MockVariableValue(&setting.Federation.Enabled, true)()
	defer test.MockVariableValue(&testWebRoutes, routers.NormalRoutes())()
	defer tests.PrepareTestEnv(t)()

	userID := 2
	username := "user2"
	req := NewRequest(t, "GET", fmt.Sprintf("/api/v1/activitypub/user-id/%v", userID))
	resp := MakeRequest(t, req, http.StatusOK)
	assert.Contains(t, resp.Body.String(), "@context")

	var person ap.Person
	err := person.UnmarshalJSON(resp.Body.Bytes())
	require.NoError(t, err)

	assert.Equal(t, ap.PersonType, person.Type)
	assert.Equal(t, username, person.PreferredUsername.String())
	keyID := person.GetID().String()
	assert.Regexp(t, fmt.Sprintf("activitypub/user-id/%v$", userID), keyID)
	assert.Regexp(t, fmt.Sprintf("activitypub/user-id/%v/outbox$", userID), person.Outbox.GetID().String())
	assert.Regexp(t, fmt.Sprintf("activitypub/user-id/%v/inbox$", userID), person.Inbox.GetID().String())

	pubKey := person.PublicKey
	assert.NotNil(t, pubKey)
	publicKeyID := keyID + "#main-key"
	assert.Equal(t, pubKey.ID.String(), publicKeyID)

	pubKeyPem := pubKey.PublicKeyPem
	assert.NotNil(t, pubKeyPem)
	assert.Regexp(t, "^-----BEGIN PUBLIC KEY-----", pubKeyPem)
}

func TestActivityPubMissingPerson(t *testing.T) {
	defer test.MockVariableValue(&setting.Federation.Enabled, true)()
	defer test.MockVariableValue(&testWebRoutes, routers.NormalRoutes())()
	defer tests.PrepareTestEnv(t)()

	req := NewRequest(t, "GET", "/api/v1/activitypub/user-id/999999999")
	resp := MakeRequest(t, req, http.StatusNotFound)
	assert.Contains(t, resp.Body.String(), "user does not exist")
}

func TestActivityPubPersonInbox(t *testing.T) {
	defer test.MockVariableValue(&setting.Federation.Enabled, true)()
	defer test.MockVariableValue(&testWebRoutes, routers.NormalRoutes())()

	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		defer test.MockVariableValue(&setting.AppURL, u.String())()
		user1 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})

		user1url := u.JoinPath("/api/v1/activitypub/user-id/1").String() + "#main-key"
		cf, err := activitypub.GetClientFactory(db.DefaultContext)
		require.NoError(t, err)
		c, err := cf.WithKeys(db.DefaultContext, user1, user1url)
		require.NoError(t, err)
		user2inboxurl := u.JoinPath("/api/v1/activitypub/user-id/2/inbox").String()

		// Signed request "succeeds"
		resp, err := c.Post([]byte{}, user2inboxurl)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotAcceptable, resp.StatusCode)

		// Unsigned request fails
		req := NewRequest(t, "POST", user2inboxurl)
		MakeRequest(t, req, http.StatusBadRequest)
	})
}
