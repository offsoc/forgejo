// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"net/http"
	"testing"

	auth_model "code.gitea.io/gitea/models/auth"
	api "code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/tests"

	"github.com/stretchr/testify/assert"
)

func TestAPIGistsSearch(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	req := NewRequest(t, "GET", "/api/v1/gists/search")
	resp := MakeRequest(t, req, http.StatusOK)

	var gists api.GistList
	DecodeJSON(t, resp, &gists)

	assert.Len(t, gists.Gists, 2)

	assert.Equal(t, int64(4), gists.Gists[0].ID)
	assert.Equal(t, int64(3), gists.Gists[0].Owner.ID)

	assert.Equal(t, int64(1), gists.Gists[1].ID)
	assert.Equal(t, int64(2), gists.Gists[1].Owner.ID)
}

func TestAPIGistsCreate(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	session := loginUser(t, "user2")
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteGist)

	newGist := &api.CreateGistOption{
		Name:        "New Gist",
		Visibility:  "public",
		Description: "New Description",
		Files: []*api.GistFile{
			{
				Name:    "new.txt",
				Content: "New text",
			},
		},
	}

	req := NewRequestWithJSON(t, "POST", "/api/v1/gists", &newGist).AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusCreated)

	var gist api.Gist
	DecodeJSON(t, resp, &gist)

	assert.Equal(t, "New Gist", gist.Name)
	assert.Equal(t, "public", gist.Visibility)
	assert.Equal(t, "New Description", gist.Description)
	assert.Equal(t, int64(2), gist.Owner.ID)
}

func TestAPIGistsGet(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	req := NewRequest(t, "GET", "/api/v1/gists/df852aec")
	resp := MakeRequest(t, req, http.StatusOK)

	var gist api.Gist
	DecodeJSON(t, resp, &gist)

	assert.Equal(t, "PublicGist", gist.Name)
	assert.Equal(t, "This is a Description", gist.Description)
	assert.Equal(t, "public", gist.Visibility)
	assert.Equal(t, int64(2), gist.Owner.ID)
}

func TestAPIGistsFiles(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	session := loginUser(t, "user2")
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteGist)

	req := NewRequest(t, "GET", "/api/v1/gists/df852aec/files")
	resp := MakeRequest(t, req, http.StatusOK)

	var files []*api.GistFile
	DecodeJSON(t, resp, &files)

	assert.Len(t, files, 1)
	assert.Equal(t, "test.txt", files[0].Name)
	assert.Equal(t, "Hello World", files[0].Content)

	newFiles := &api.UpdateGistFilesOption{
		Files: []*api.GistFile{
			{
				Name:    "new.txt",
				Content: "New text",
			},
		},
	}

	MakeRequest(t, NewRequestWithJSON(t, "POST", "/api/v1/gists/df852aec/files", &newFiles).AddTokenAuth(token), http.StatusNoContent)

	req = NewRequest(t, "GET", "/api/v1/gists/df852aec/files")
	resp = MakeRequest(t, req, http.StatusOK)

	DecodeJSON(t, resp, &files)

	assert.Len(t, files, 1)
	assert.Equal(t, "new.txt", files[0].Name)
	assert.Equal(t, "New text", files[0].Content)
}

func TestAPIGistsDelete(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	session := loginUser(t, "user2")
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteGist)

	MakeRequest(t, NewRequest(t, "DELETE", "/api/v1/gists/df852aec").AddTokenAuth(token), http.StatusNoContent)
}
