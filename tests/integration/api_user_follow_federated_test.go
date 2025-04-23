// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"testing"

	auth_model "forgejo.org/models/auth"
	"forgejo.org/models/forgefed"
	"forgejo.org/models/unittest"
	"forgejo.org/models/user"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/structs"
	"forgejo.org/modules/test"
	"forgejo.org/routers"
	"forgejo.org/tests"
	"github.com/stretchr/testify/assert"
)

// Flow of this test is documeted at: https://codeberg.org/meissa/federation/src/branch/federated-user-activity-following/doc/user-activity-following.md
func TestAPIFollowFederated(t *testing.T) {
	defer test.MockVariableValue(&setting.Federation.Enabled, true)()
	defer test.MockVariableValue(&testWebRoutes, routers.NormalRoutes())()
	defer tests.PrepareTestEnv(t)()

	mock := test.NewFederationServerMock()
	federatedSrv := mock.DistantServer(t)
	defer federatedSrv.Close()

	user10 := "user10"

	session10 := loginUser(t, user10)
	token10 := getTokenForLoggedInUser(t, session10, auth_model.AccessTokenScopeWriteUser)

	t.Run("Follow", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequestWithJSON(t, "POST",
			"/api/v1/user/activitypub/follow",
			&structs.APRemoteFollowOption{
				Target: fmt.Sprintf("%s/api/v1/activitypub/user-id/15", federatedSrv.URL),
			}).
			AddTokenAuth(token10)
		MakeRequest(t, req, http.StatusNoContent)
		federationHost := unittest.AssertExistsAndLoadBean(t, &forgefed.FederationHost{HostFqdn: "127.0.0.1"})
		unittest.AssertExistsAndLoadBean(t, &user.FederatedUser{ExternalID: "15", FederationHostID: federationHost.ID})
		assert.Contains(t, mock.LastPost, "\"target\":\"http://DISTANT_FEDERATION_HOST/api/v1/activitypub/user-id/15\"")
		assert.Contains(t, mock.LastPost, "\"object\":\"http://DISTANT_FEDERATION_HOST/api/v1/activitypub/user-id/15\"")
	})
}
