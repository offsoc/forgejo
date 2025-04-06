// Copyright 2022 The Gitea Authors. All rights reserved.
// Copyright 2023 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package activitypub

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"forgejo.org/models/db"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCurrentTime(t *testing.T) {
	date := CurrentTime()
	_, err := time.Parse(http.TimeFormat, date)
	require.NoError(t, err)
	assert.Equal(t, "GMT", date[len(date)-3:])
}

/* ToDo: Set Up tests for http get requests

Set up an expected response for GET on api with user-id = 1:
{
  "@context": [
    "https://www.w3.org/ns/activitystreams",
    "https://w3id.org/security/v1"
  ],
  "id": "http://localhost:3000/api/v1/activitypub/user-id/1",
  "type": "Person",
  "icon": {
    "type": "Image",
    "mediaType": "image/png",
    "url": "http://localhost:3000/avatar/3120fd0edc57d5d41230013ad88232e2"
  },
  "url": "http://localhost:3000/me",
  "inbox": "http://localhost:3000/api/v1/activitypub/user-id/1/inbox",
  "outbox": "http://localhost:3000/api/v1/activitypub/user-id/1/outbox",
  "preferredUsername": "me",
  "publicKey": {
    "id": "http://localhost:3000/api/v1/activitypub/user-id/1#main-key",
    "owner": "http://localhost:3000/api/v1/activitypub/user-id/1",
    "publicKeyPem": "-----BEGIN PUBLIC KEY-----\nMIIBojANBgkqhkiG9w0BAQEFAAOCAY8AMIIBigKCAYEAo1VDZGWQBDTWKhpWiPQp\n7nD94UsKkcoFwDQVuxE3bMquKEHBomB4cwUnVou922YkL3AmSOr1sX2yJQGqnCLm\nOeKS74/mCIAoYlu0d75bqY4A7kE2VrQmQLZBbmpCTfrPqDaE6Mfm/kXaX7+hsrZS\n4bVvzZCYq8sjtRxdPk+9ku2QhvznwTRlWLvwHmFSGtlQYPRu+f/XqoVM/DVRA/Is\nwDk9yiNIecV+Isus0CBq1jGQkfuVNu1GK2IvcSg9MoDm3VH/tCayAP+xWm0g7sC8\nKay6Y/khvTvE7bWEKGQsJGvi3+4wITLVLVt+GoVOuCzdbhTV2CHBzn7h30AoZD0N\nY6eyb+Q142JykoHadcRwh1a36wgoG7E496wPvV3ST8xdiClca8cDNhOzCj8woY+t\nTFCMl32U3AJ4e/cAsxKRocYLZqc95dDqdNQiIyiRMMkf5NaA/QvelY4PmFuHC0WR\nVuJ4A3mcti2QLS9j0fSwSJdlfolgW6xaPgjdvuSQsgX1AgMBAAE=\n-----END PUBLIC KEY-----\n"
  }
}

Set up a user called "me" for all tests



*/

func TestClientCtx(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
	pubID := "myGpgId"
	cf, err := NewClientFactory()
	log.Debug("ClientFactory: %v\nError: %v", cf, err)
	require.NoError(t, err)

	c, err := cf.WithKeys(db.DefaultContext, user, pubID)

	log.Debug("Client: %v\nError: %v", c, err)
	require.NoError(t, err)
	_ = NewContext(db.DefaultContext, cf)
}

/* TODO: bring this test to work or delete
func TestActivityPubSignedGet(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1, Name: "me"})
	pubID := "myGpgId"
	c, err := NewClient(db.DefaultContext, user, pubID)
	require.NoError(t, err)

	expected := "TestActivityPubSignedGet"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Regexp(t, regexp.MustCompile("^"+setting.Federation.DigestAlgorithm), r.Header.Get("Digest"))
		assert.Contains(t, r.Header.Get("Signature"), pubID)
		assert.Equal(t, r.Header.Get("Content-Type"), ActivityStreamsContentType)
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Equal(t, expected, string(body))
		fmt.Fprint(w, expected)
	}))
	defer srv.Close()

	r, err := c.Get(srv.URL)
	require.NoError(t, err)
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	require.NoError(t, err)
	assert.Equal(t, expected, string(body))

}
*/

func TestActivityPubSignedPost(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
	pubID := "https://example.com/pubID"
	cf, err := NewClientFactory()
	require.NoError(t, err)
	c, err := cf.WithKeys(db.DefaultContext, user, pubID)
	require.NoError(t, err)

	expected := "BODY"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Regexp(t, "^"+setting.Federation.DigestAlgorithm, r.Header.Get("Digest"))
		assert.Contains(t, r.Header.Get("Signature"), pubID)
		assert.Equal(t, ActivityStreamsContentType, r.Header.Get("Content-Type"))
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Equal(t, expected, string(body))
		fmt.Fprint(w, expected)
	}))
	defer srv.Close()

	r, err := c.Post([]byte(expected), srv.URL)
	require.NoError(t, err)
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	require.NoError(t, err)
	assert.Equal(t, expected, string(body))
}
