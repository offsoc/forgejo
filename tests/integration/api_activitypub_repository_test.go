// Copyright 2024, 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"forgejo.org/models/db"
	"forgejo.org/models/forgefed"
	"forgejo.org/models/unittest"
	"forgejo.org/models/user"
	"forgejo.org/modules/activitypub"
	forgefed_modules "forgejo.org/modules/forgefed"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/test"
	"forgejo.org/routers"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActivityPubRepository(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	defer test.MockVariableValue(&setting.Federation.Enabled, true)()
	defer test.MockVariableValue(&testWebRoutes, routers.NormalRoutes())()

	mock := test.NewFederationServerMock()
	federatedSrv := mock.DistantServer(t)
	defer federatedSrv.Close()

	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		repositoryID := 2

		cf, err := activitypub.GetClientFactory(db.DefaultContext)
		require.NoError(t, err)

		c, err := cf.WithKeysDirect(db.DefaultContext, mock.Persons[0].PrivKey,
			mock.Persons[0].KeyID(federatedSrv.URL))
		require.NoError(t, err)

		resp, err := c.GetBody(fmt.Sprintf("%sapi/v1/activitypub/repository-id/%d", u, repositoryID))
		require.NoError(t, err)
		assert.Contains(t, string(resp), "@context")

		var repository forgefed_modules.Repository
		err = repository.UnmarshalJSON(resp)
		require.NoError(t, err)

		assert.Regexp(t, fmt.Sprintf("activitypub/repository-id/%d$", repositoryID), repository.GetID().String())
	})
}

func TestActivityPubMissingRepository(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	defer test.MockVariableValue(&setting.Federation.Enabled, true)()
	defer test.MockVariableValue(&setting.Federation.SignatureEnforced, false)()
	defer test.MockVariableValue(&testWebRoutes, routers.NormalRoutes())()

	repositoryID := 9999999
	req := NewRequest(t, "GET", fmt.Sprintf("/api/v1/activitypub/repository-id/%d", repositoryID))
	resp := MakeRequest(t, req, http.StatusNotFound)
	assert.Contains(t, resp.Body.String(), "repository does not exist")
}

func TestActivityPubRepositoryInboxValid(t *testing.T) {
	defer test.MockVariableValue(&setting.Federation.Enabled, true)()
	defer test.MockVariableValue(&testWebRoutes, routers.NormalRoutes())()

	mock := test.NewFederationServerMock()
	federatedSrv := mock.DistantServer(t)
	defer federatedSrv.Close()

	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		repositoryID := 2
		timeNow := time.Now().UTC()

		cf, err := activitypub.GetClientFactory(db.DefaultContext)
		require.NoError(t, err)

		c, err := cf.WithKeysDirect(db.DefaultContext, mock.Persons[0].PrivKey,
			mock.Persons[0].KeyID(federatedSrv.URL))
		require.NoError(t, err)

		repoInboxURL := u.JoinPath(fmt.Sprintf("/api/v1/activitypub/repository-id/%d/inbox", repositoryID)).String()

		activity1 := []byte(fmt.Sprintf(
			`{"type":"Like",`+
				`"startTime":"%s",`+
				`"actor":"%s/api/v1/activitypub/user-id/15",`+
				`"object":"%s"}`,
			timeNow.Format(time.RFC3339),
			federatedSrv.URL, u.JoinPath(fmt.Sprintf("/api/v1/activitypub/repository-id/%d", repositoryID)).String()))
		t.Logf("activity: %s", activity1)
		resp, err := c.Post(activity1, repoInboxURL)

		require.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, resp.StatusCode)

		federationHost := unittest.AssertExistsAndLoadBean(t, &forgefed.FederationHost{HostFqdn: "127.0.0.1"})
		federatedUser := unittest.AssertExistsAndLoadBean(t, &user.FederatedUser{ExternalID: "15", FederationHostID: federationHost.ID})
		unittest.AssertExistsAndLoadBean(t, &user.User{ID: federatedUser.UserID})

		// A like activity by a different user of the same federated host.
		activity2 := []byte(fmt.Sprintf(
			`{"type":"Like",`+
				`"startTime":"%s",`+
				`"actor":"%s/api/v1/activitypub/user-id/30",`+
				`"object":"%s"}`,
			// Make sure this activity happens later then the one before
			timeNow.Add(time.Second).Format(time.RFC3339),
			federatedSrv.URL, u.JoinPath(fmt.Sprintf("/api/v1/activitypub/repository-id/%d", repositoryID)).String()))
		t.Logf("activity: %s", activity2)
		resp, err = c.Post(activity2, repoInboxURL)

		require.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, resp.StatusCode)

		federatedUser = unittest.AssertExistsAndLoadBean(t, &user.FederatedUser{ExternalID: "30", FederationHostID: federationHost.ID})
		unittest.AssertExistsAndLoadBean(t, &user.User{ID: federatedUser.UserID})

		// The same user sends another like activity
		otherRepositoryID := 3
		otherRepoInboxURL := u.JoinPath(fmt.Sprintf("/api/v1/activitypub/repository-id/%d/inbox", otherRepositoryID)).String()
		activity3 := []byte(fmt.Sprintf(
			`{"type":"Like",`+
				`"startTime":"%s",`+
				`"actor":"%s/api/v1/activitypub/user-id/30",`+
				`"object":"%s"}`,
			// Make sure this activity happens later then the ones before
			timeNow.Add(time.Second*2).Format(time.RFC3339),
			federatedSrv.URL, u.JoinPath(fmt.Sprintf("/api/v1/activitypub/repository-id/%d", otherRepositoryID)).String()))
		t.Logf("activity: %s", activity3)
		resp, err = c.Post(activity3, otherRepoInboxURL)

		require.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, resp.StatusCode)

		federatedUser = unittest.AssertExistsAndLoadBean(t, &user.FederatedUser{ExternalID: "30", FederationHostID: federationHost.ID})
		unittest.AssertExistsAndLoadBean(t, &user.User{ID: federatedUser.UserID})

		// Replay activity2.
		resp, err = c.Post(activity2, repoInboxURL)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotAcceptable, resp.StatusCode)
	})
}

func TestActivityPubRepositoryInboxInvalid(t *testing.T) {
	defer test.MockVariableValue(&setting.Federation.Enabled, true)()
	defer test.MockVariableValue(&setting.Federation.SignatureEnforced, false)()
	defer test.MockVariableValue(&testWebRoutes, routers.NormalRoutes())()

	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		apServerActor := user.NewAPServerActor()
		repositoryID := 2

		cf, err := activitypub.GetClientFactory(db.DefaultContext)
		require.NoError(t, err)

		c, err := cf.WithKeys(db.DefaultContext, apServerActor, apServerActor.KeyID())
		require.NoError(t, err)

		repoInboxURL := u.JoinPath(fmt.Sprintf("/api/v1/activitypub/repository-id/%d/inbox", repositoryID)).String()
		activity := []byte(`{"type":"Wrong"}`)
		resp, err := c.Post(activity, repoInboxURL)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotAcceptable, resp.StatusCode)
	})
}
