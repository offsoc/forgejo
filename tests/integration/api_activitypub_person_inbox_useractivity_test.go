// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"forgejo.org/models/activities"
	auth_model "forgejo.org/models/auth"
	"forgejo.org/models/db"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/activitypub"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/structs"
	api "forgejo.org/modules/structs"
	"forgejo.org/modules/test"
	"forgejo.org/routers"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Flow of this test is documented at: https://codeberg.org/forgejo-contrib/federation/src/branch/main/doc/user-activity-following.md
func TestActivityPubPersonInboxNoteFromDistant(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	defer test.MockVariableValue(&setting.Federation.Enabled, true)()
	defer test.MockVariableValue(&setting.Federation.SignatureEnforced, false)()
	defer test.MockVariableValue(&testWebRoutes, routers.NormalRoutes())()

	mock := test.NewFederationServerMock()
	federatedSrv := mock.DistantServer(t)
	defer federatedSrv.Close()

	onGiteaRun(t, func(t *testing.T, localUrl *url.URL) {
		defer test.MockVariableValue(&setting.AppURL, localUrl.String())()

		distantUrl := federatedSrv.URL
		distantUser15URL := fmt.Sprintf("%s/api/v1/activitypub/user-id/15", distantUrl)

		localUser2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
		localUser2URL := localUrl.JoinPath("/api/v1/activitypub/user-id/2").String()
		localUser2Inbox := localUrl.JoinPath("/api/v1/activitypub/user-id/2/inbox").String()
		localSession2 := loginUser(t, localUser2.LoginName)
		localSecssion2Token := getTokenForLoggedInUser(t, localSession2, auth_model.AccessTokenScopeWriteUser)

		// follow (local follows distant)
		req := NewRequestWithJSON(t, "POST",
			"/api/v1/user/activitypub/follow",
			&structs.APRemoteFollowOption{
				Target: distantUser15URL,
			}).
			AddTokenAuth(localSecssion2Token)
		MakeRequest(t, req, http.StatusNoContent)

		// send note (distant -> local)
		distantNoteUrl := fmt.Sprintf("%s/api/v1/activitypub/note/104", distantUrl)
		userActivity := []byte(fmt.Sprintf(
			`{"type":"Create",`+
				`"actor":"%s",`+
				`"to": ["https://www.w3.org/ns/activitystreams#Public"],`+
				`"cc": ["%s"],`+
				`"object": {"type":"Note","content":"The Content!",`+
				`"url":"%s"}}`,
			distantUser15URL,
			localUser2URL,
			distantNoteUrl,
		))
		cf, err := activitypub.GetClientFactory(db.DefaultContext)
		require.NoError(t, err)
		c, err := cf.WithKeysDirect(db.DefaultContext, mock.ApActor.PrivKey,
			mock.ApActor.APActorKeyID(federatedSrv.URL))
		require.NoError(t, err)
		resp, err := c.Post(userActivity, localUser2Inbox)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, resp.StatusCode)

		// check user activity exists on local
		unittest.AssertExistsAndLoadBean(t, &activities.FederatedUserActivity{NoteURL: distantNoteUrl})
	})
}

func TestActivityPubPersonInboxNoteToDistant(t *testing.T) {
	defer tests.AddFixtures("tests/integration/fixtures/TestActivityPubPersonInboxNoteToDistant")()
	defer tests.PrepareTestEnv(t)()
	defer test.MockVariableValue(&setting.Federation.Enabled, true)()
	defer test.MockVariableValue(&setting.Federation.SignatureEnforced, false)()
	defer test.MockVariableValue(&testWebRoutes, routers.NormalRoutes())()

	mock := test.NewFederationServerMock()
	federatedSrv := mock.DistantServer(t)
	defer federatedSrv.Close()

	onGiteaRun(t, func(t *testing.T, localUrl *url.URL) {
		defer test.MockVariableValue(&setting.AppURL, localUrl.String())()

		distantUrl := federatedSrv.URL
		distantUser15URL := fmt.Sprintf("%s/api/v1/activitypub/user-id/15", distantUrl)

		localUser2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
		localUser2URL := localUrl.JoinPath("/api/v1/activitypub/user-id/2").String()
		localUser2Inbox := localUrl.JoinPath("/api/v1/activitypub/user-id/2/inbox").String()
		localSession2 := loginUser(t, localUser2.LoginName)
		localSecssion2Token := getTokenForLoggedInUser(t, localSession2, auth_model.AccessTokenScopeWriteIssue)
		println(localSecssion2Token)

		repo, _, f := tests.CreateDeclarativeRepoWithOptions(t, localUser2, tests.DeclarativeRepoOptions{})
		defer f()

		// follow (distant follows local)
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

		// local action which triggers an user activity
		IssueURL := fmt.Sprintf("/api/v1/repos/%s/issues?state=all", repo.FullName())
		req := NewRequestWithJSON(t, "POST", IssueURL, &api.CreateIssueOption{
			Title: "ActivityFeed test",
			Body:  "Nothing to see here!",
		}).AddTokenAuth(localSecssion2Token)
		MakeRequest(t, req, http.StatusCreated)

		// check for activity on distant inbox
		assert.Equal(t, "", mock.LastPost)
	})
}
