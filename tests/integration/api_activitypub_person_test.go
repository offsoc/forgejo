// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"forgejo.org/models/db"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/activitypub"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/test"
	"forgejo.org/routers"
	"forgejo.org/tests"

	ap "github.com/go-ap/activitypub"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActivityPubPerson(t *testing.T) {
	defer test.MockVariableValue(&setting.Federation.Enabled, true)()
	defer test.MockVariableValue(&testWebRoutes, routers.NormalRoutes())()
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		userID := 2
		username := "user2"
		userURL := fmt.Sprintf("%sapi/v1/activitypub/user-id/%d", u, userID)

		user1 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})

		clientFactory, err := activitypub.GetClientFactory(db.DefaultContext)
		require.NoError(t, err)

		apClient, err := clientFactory.WithKeys(db.DefaultContext, user1, user1.APActorKeyID())
		require.NoError(t, err)

		// Unsigned request
		t.Run("UnsignedRequest", func(t *testing.T) {
			req := NewRequest(t, "GET", userURL)
			MakeRequest(t, req, http.StatusBadRequest)
		})

		t.Run("SignedRequestValidation", func(t *testing.T) {
			// Signed request
			resp, err := apClient.GetBody(userURL)
			require.NoError(t, err)

			var person ap.Person
			err = person.UnmarshalJSON(resp)
			require.NoError(t, err)

			assert.Equal(t, ap.PersonType, person.Type)
			assert.Equal(t, username, person.PreferredUsername.String())
			assert.Regexp(t, fmt.Sprintf("activitypub/user-id/%d$", userID), person.GetID())
			assert.Regexp(t, fmt.Sprintf("activitypub/user-id/%d/outbox$", userID), person.Outbox.GetID().String())
			assert.Regexp(t, fmt.Sprintf("activitypub/user-id/%d/inbox$", userID), person.Inbox.GetID().String())

			assert.NotNil(t, person.PublicKey)
			assert.Regexp(t, fmt.Sprintf("activitypub/user-id/%d#main-key$", userID), person.PublicKey.ID)

			assert.NotNil(t, person.PublicKey.PublicKeyPem)
			assert.Regexp(t, "^-----BEGIN PUBLIC KEY-----", person.PublicKey.PublicKeyPem)
		})
	})
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

		// Signed request succeeds
		resp, err := c.Post([]byte{}, user2inboxurl)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, resp.StatusCode)

		// Unsigned request fails
		req := NewRequest(t, "POST", user2inboxurl)
		MakeRequest(t, req, http.StatusBadRequest)
	})
}
