// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"testing"

	auth_model "forgejo.org/models/auth"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIDownloadArchive(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	session := loginUser(t, user2.LowerName)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeReadRepository)

	link, _ := url.Parse(fmt.Sprintf("/api/v1/repos/%s/%s/archive/master.zip", user2.Name, repo.Name))
	resp := MakeRequest(t, NewRequest(t, "GET", link.String()).AddTokenAuth(token), http.StatusOK)
	bs, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Len(t, bs, 320)
	assert.Equal(t, "application/zip", resp.Header().Get("Content-Type"))

	link, _ = url.Parse(fmt.Sprintf("/api/v1/repos/%s/%s/archive/master.tar.gz", user2.Name, repo.Name))
	resp = MakeRequest(t, NewRequest(t, "GET", link.String()).AddTokenAuth(token), http.StatusOK)
	bs, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Len(t, bs, 266)
	assert.Equal(t, "application/gzip", resp.Header().Get("Content-Type"))

	// Must return a link to a commit ID as the "immutable" archive link
	linkHeaderRe := regexp.MustCompile(`<(?P<url>https?://.*/api/v1/repos/user2/repo1/archive/[a-f0-9]+\.tar\.gz.*)>; rel="immutable"`)
	m := linkHeaderRe.FindStringSubmatch(resp.Header().Get("Link"))
	assert.NotEmpty(t, m[1])
	resp = MakeRequest(t, NewRequest(t, "GET", m[1]).AddTokenAuth(token), http.StatusOK)
	bs2, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	// The locked URL should give the same bytes as the non-locked one
	assert.Equal(t, bs, bs2)

	link, _ = url.Parse(fmt.Sprintf("/api/v1/repos/%s/%s/archive/master.bundle", user2.Name, repo.Name))
	resp = MakeRequest(t, NewRequest(t, "GET", link.String()).AddTokenAuth(token), http.StatusOK)
	bs, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Len(t, bs, 382)
	assert.Equal(t, "application/octet-stream", resp.Header().Get("Content-Type"))

	link, _ = url.Parse(fmt.Sprintf("/api/v1/repos/%s/%s/archive/master", user2.Name, repo.Name))
	MakeRequest(t, NewRequest(t, "GET", link.String()).AddTokenAuth(token), http.StatusBadRequest)
}

func TestAPIDownloadArchive2(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	session := loginUser(t, user2.LowerName)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeReadRepository)

	link, _ := url.Parse(fmt.Sprintf("/api/v1/repos/%s/%s/zipball/master", user2.Name, repo.Name))
	resp := MakeRequest(t, NewRequest(t, "GET", link.String()).AddTokenAuth(token), http.StatusOK)
	bs, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Len(t, bs, 320)

	link, _ = url.Parse(fmt.Sprintf("/api/v1/repos/%s/%s/tarball/master", user2.Name, repo.Name))
	resp = MakeRequest(t, NewRequest(t, "GET", link.String()).AddTokenAuth(token), http.StatusOK)
	bs, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Len(t, bs, 266)

	// Must return a link to a commit ID as the "immutable" archive link
	linkHeaderRe := regexp.MustCompile(`^<(https?://.*/api/v1/repos/user2/repo1/archive/[a-f0-9]+\.tar\.gz.*)>; rel="immutable"$`)
	m := linkHeaderRe.FindStringSubmatch(resp.Header().Get("Link"))
	assert.NotEmpty(t, m[1])
	resp = MakeRequest(t, NewRequest(t, "GET", m[1]).AddTokenAuth(token), http.StatusOK)
	bs2, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	// The locked URL should give the same bytes as the non-locked one
	assert.Equal(t, bs, bs2)

	link, _ = url.Parse(fmt.Sprintf("/api/v1/repos/%s/%s/bundle/master", user2.Name, repo.Name))
	resp = MakeRequest(t, NewRequest(t, "GET", link.String()).AddTokenAuth(token), http.StatusOK)
	bs, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Len(t, bs, 382)

	link, _ = url.Parse(fmt.Sprintf("/api/v1/repos/%s/%s/archive/master", user2.Name, repo.Name))
	MakeRequest(t, NewRequest(t, "GET", link.String()).AddTokenAuth(token), http.StatusBadRequest)
}
