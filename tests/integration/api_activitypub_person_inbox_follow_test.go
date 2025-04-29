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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Flow of this test is documented at: https://codeberg.org/forgejo-contrib/federation/src/branch/main/doc/user-activity-following.md
func TestActivityPubPersonInboxFollow(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	defer test.MockVariableValue(&setting.Federation.Enabled, true)()
	defer test.MockVariableValue(&setting.Federation.SignatureEnforced, false)()
	defer test.MockVariableValue(&testWebRoutes, routers.NormalRoutes())()

	mock := test.NewFederationServerMock()
	federatedSrv := mock.DistantServer(t)
	defer federatedSrv.Close()

	onGiteaRun(t, func(t *testing.T, localUrl *url.URL) {
		defer test.MockVariableValue(&setting.AppURL, localUrl.String())()

		distantURL := federatedSrv.URL
		distantUser15URL := fmt.Sprintf("%s/api/v1/activitypub/user-id/15", distantURL)

		localUser := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
		localUser2URL := localUrl.JoinPath("/api/v1/activitypub/user-id/2").String()
		localUser2Inbox := localUrl.JoinPath("/api/v1/activitypub/user-id/2/inbox").String()

		// distant follows local
		followActivity := []byte(fmt.Sprintf(
			`{"type":"Follow",`+
				`"actor":"%s",`+
				`"object":"%s"}`,
			distantUser15URL,
			localUser2URL,
		))
		cf, err := activitypub.GetClientFactory(db.DefaultContext)
		require.NoError(t, err)
		c, err := cf.WithKeysDirect(db.DefaultContext, mock.ApActor.PrivKey,
			mock.ApActor.APActorKeyID(federatedSrv.URL))
		require.NoError(t, err)
		resp, err := c.Post(followActivity, localUser2Inbox)
		require.NoError(t, err)
		assert.Equal(t, http.StatusAccepted, resp.StatusCode)

		// local follow exists
		distantFederatedUser := unittest.AssertExistsAndLoadBean(t, &user_model.FederatedUser{ExternalID: "15"})
		unittest.AssertExistsAndLoadBean(t,
			&user_model.FederatedUserFollower{
				FollowedUserID:  localUser.ID,
				FollowingUserID: distantFederatedUser.UserID,
			},
		)

		// distant is informed about accepting follow
		assert.Contains(t, mock.LastPost, "\"type\":\"Accept\"")

		// distant undoes follow
		undoFollowActivity := []byte(fmt.Sprintf(
			`{"type":"Undo",`+
				`"actor":"%s",`+
				`"object":{"type":"Follow",`+
				`"actor":"%s",`+
				`"object":"%s"}}`,
			distantUser15URL,
			distantUser15URL,
			localUser2URL,
		))
		c, err = cf.WithKeysDirect(db.DefaultContext, mock.ApActor.PrivKey,
			mock.ApActor.APActorKeyID(federatedSrv.URL))
		require.NoError(t, err)
		resp, err = c.Post(undoFollowActivity, localUser2Inbox)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, resp.StatusCode)

		// local follow removed
		unittest.AssertNotExistsBean(t,
			&user_model.FederatedUserFollower{
				FollowedUserID:  localUser.ID,
				FollowingUserID: distantFederatedUser.UserID,
			},
		)
	})
}
