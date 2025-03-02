// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
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

func TestActivityPubClientBodySize(t *testing.T) {
	defer test.MockVariableValue(&setting.Federation.Enabled, true)()
	defer test.MockVariableValue(&testWebRoutes, routers.NormalRoutes())()

	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		user1 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})

		clientFactory, err := activitypub.GetClientFactory(db.DefaultContext)
		require.NoError(t, err)

		apClient, err := clientFactory.WithKeys(db.DefaultContext, user1, user1.APActorKeyID())
		require.NoError(t, err)

		url := u.JoinPath("/api/v1/nodeinfo").String()

		// Request with normal MaxSize
		t.Run("NormalMaxSize", func(t *testing.T) {
			resp, err := apClient.GetBody(url)
			require.NoError(t, err)
			assert.Contains(t, string(resp), "forgejo")
		})

		// Set MaxSize to something very low to always fail
		// Request with low MaxSize
		t.Run("LowMaxSize", func(t *testing.T) {
			defer test.MockVariableValue(&setting.Federation.MaxSize, 100)()

			_, err = apClient.GetBody(url)
			require.Error(t, err)
			assert.ErrorContains(t, err, "Request returned")
		})
	})
}
