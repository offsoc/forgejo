// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"testing"

	auth_model "code.gitea.io/gitea/models/auth"
	"code.gitea.io/gitea/models/forgefed"
	"code.gitea.io/gitea/models/unittest"
	"code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/modules/test"
	"code.gitea.io/gitea/routers"
	"code.gitea.io/gitea/tests"
	ttools "code.gitea.io/gitea/tests/tools"
)

func TestAPIFollowFederated(t *testing.T) {
	defer test.MockVariableValue(&setting.Federation.Enabled, true)()
	defer test.MockVariableValue(&testWebRoutes, routers.NormalRoutes())()
	defer tests.PrepareTestEnv(t)()

	mock := ttools.NewFederationServerMock()
	federatedSrv := mock.DistantServer(t)
	defer federatedSrv.Close()

	//user1 := "user4"
	user2 := "user10"

	//session1 := loginUser(t, user1)
	//token1 := getTokenForLoggedInUser(t, session1, auth_model.AccessTokenScopeReadUser)

	session2 := loginUser(t, user2)
	token2 := getTokenForLoggedInUser(t, session2, auth_model.AccessTokenScopeWriteUser)

	t.Run("Follow", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequestWithJSON(t, "POST",
			"/api/v1/user/activitypub/follow",
			&structs.APRemoteFollowOption{
				Target: fmt.Sprintf("%s/api/v1/activitypub/user-id/15", federatedSrv.URL),
			}).
			AddTokenAuth(token2)
		MakeRequest(t, req, http.StatusNoContent)
		federationHost := unittest.AssertExistsAndLoadBean(t, &forgefed.FederationHost{HostFqdn: "127.0.0.1"})
		followedUser := unittest.AssertExistsAndLoadBean(t, &user.User{Name: user2})
		followingFederatedUser := unittest.AssertExistsAndLoadBean(t, &user.FederatedUser{ExternalID: "15", FederationHostID: federationHost.ID})
		unittest.AssertExistsAndLoadBean(t, &user.FederatedUserFollower{
			FollowedUserID:  followedUser.ID,
			FollowingUserID: followingFederatedUser.ID,
		})
	})
}
