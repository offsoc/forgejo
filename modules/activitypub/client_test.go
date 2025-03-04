// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package activitypub

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/setting"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActivityPubSignedPost(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
	pubID := "https://example.com/pubID"
	c, err := NewClient(db.DefaultContext, user, pubID)
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
