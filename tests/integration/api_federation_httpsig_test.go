// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"forgejo.org/models/db"
	"forgejo.org/models/forgefed"
	"forgejo.org/models/unittest"
	"forgejo.org/models/user"
	"forgejo.org/modules/activitypub"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/test"
	"forgejo.org/routers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFederationHttpSigValidation(t *testing.T) {
	defer test.MockVariableValue(&setting.Federation.Enabled, true)()
	defer test.MockVariableValue(&testWebRoutes, routers.NormalRoutes())()

	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		userID := 2
		userURL := fmt.Sprintf("%sapi/v1/activitypub/user-id/%d", u, userID)

		user1 := unittest.AssertExistsAndLoadBean(t, &user.User{ID: 1})

		clientFactory, err := activitypub.GetClientFactory(db.DefaultContext)
		require.NoError(t, err)

		apClient, err := clientFactory.WithKeys(db.DefaultContext, user1, user1.APActorKeyID())
		require.NoError(t, err)

		// Unsigned request
		t.Run("UnsignedRequest", func(t *testing.T) {
			req := NewRequest(t, "GET", userURL)
			MakeRequest(t, req, http.StatusBadRequest)
		})

		// Signed request
		t.Run("SignedRequest", func(t *testing.T) {
			resp, err := apClient.Get(userURL)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
		})

		// HACK HACK HACK: the host part of the URL gets set to which IP forgejo is
		// listening on, NOT localhost, which is the Domain given to forgejo which
		// is then used for eg. the keyID all requests
		applicationKeyID := fmt.Sprintf("%sapi/v1/activitypub/actor#main-key", setting.AppURL)
		actorKeyID := fmt.Sprintf("%sapi/v1/activitypub/user-id/1#main-key", setting.AppURL)

		// Check for cached public keys
		t.Run("ValidateCaches", func(t *testing.T) {
			host, err := forgefed.FindFederationHostByKeyID(db.DefaultContext, applicationKeyID)
			require.NoError(t, err)
			assert.NotNil(t, host)
			assert.True(t, host.PublicKey.Valid)

			_, user, err := user.FindFederatedUserByKeyID(db.DefaultContext, actorKeyID)
			require.NoError(t, err)
			assert.NotNil(t, user)
			assert.True(t, user.PublicKey.Valid)
		})

		// Disable signature validation
		defer test.MockVariableValue(&setting.Federation.SignatureEnforced, false)()

		// Unsigned request
		t.Run("SignatureValidationDisabled", func(t *testing.T) {
			req := NewRequest(t, "GET", userURL)
			MakeRequest(t, req, http.StatusOK)
		})
	})
}
