// Copyright 2025 The Forgejo Authors. All rights reserved.
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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFederationHttpSigValidation(t *testing.T) {
	defer test.MockVariableValue(&setting.Federation.Enabled, true)()
	defer test.MockVariableValue(&testWebRoutes, routers.NormalRoutes())()

	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		userID := 2
		userURL := fmt.Sprintf("%vapi/v1/activitypub/user-id/%v", u, userID)

		user1 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})

		clientFactory, err := activitypub.GetClientFactory(db.DefaultContext)
		require.NoError(t, err)

		apClient, err := clientFactory.WithKeys(db.DefaultContext, user1, user1.APActorKeyID())
		require.NoError(t, err)

		// Unsigned request
		req := NewRequest(t, "GET", userURL)
		MakeRequest(t, req, http.StatusBadRequest)

		// Signed request
		resp, err := apClient.Get(userURL)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Disable signature validation
		defer test.MockVariableValue(&setting.Federation.SignatureEnforced, false)()

		// Unsigned request
		req = NewRequest(t, "GET", userURL)
		MakeRequest(t, req, http.StatusOK)
	})
}
