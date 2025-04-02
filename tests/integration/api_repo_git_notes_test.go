// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	auth_model "forgejo.org/models/auth"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	api "forgejo.org/modules/structs"

	"github.com/stretchr/testify/assert"
)

func TestAPIReposGetGitNotes(t *testing.T) {
	onGiteaRun(t, func(*testing.T, *url.URL) {
		user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
		// Login as User2.
		session := loginUser(t, user.Name)
		token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeReadRepository)

		// check invalid requests
		req := NewRequestf(t, "GET", "/api/v1/repos/%s/repo1/git/notes/12345", user.Name).
			AddTokenAuth(token)
		MakeRequest(t, req, http.StatusNotFound)

		req = NewRequestf(t, "GET", "/api/v1/repos/%s/repo1/git/notes/..", user.Name).
			AddTokenAuth(token)
		MakeRequest(t, req, http.StatusUnprocessableEntity)

		// check valid request
		req = NewRequestf(t, "GET", "/api/v1/repos/%s/repo1/git/notes/65f1bf27bc3bf70f64657658635e66094edbcb4d", user.Name).
			AddTokenAuth(token)
		resp := MakeRequest(t, req, http.StatusOK)

		var apiData api.Note
		DecodeJSON(t, resp, &apiData)
		assert.Equal(t, "This is a test note\n", apiData.Message)
		assert.NotEmpty(t, apiData.Commit.Files)
		assert.NotNil(t, apiData.Commit.RepoCommit.Verification)
	})
}

func TestAPIReposSetGitNotes(t *testing.T) {
	onGiteaRun(t, func(*testing.T, *url.URL) {
		repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
		user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})

		session := loginUser(t, user.Name)
		token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository)

		req := NewRequestf(t, "GET", "/api/v1/repos/%s/git/notes/65f1bf27bc3bf70f64657658635e66094edbcb4d", repo.FullName())
		resp := MakeRequest(t, req, http.StatusOK)
		var apiData api.Note
		DecodeJSON(t, resp, &apiData)
		assert.Equal(t, "This is a test note\n", apiData.Message)

		req = NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/repos/%s/git/notes/65f1bf27bc3bf70f64657658635e66094edbcb4d", repo.FullName()), &api.NoteOptions{
			Message: "This is a new note",
		}).AddTokenAuth(token)
		resp = MakeRequest(t, req, http.StatusOK)
		DecodeJSON(t, resp, &apiData)
		assert.Equal(t, "This is a new note\n", apiData.Message)

		req = NewRequestf(t, "GET", "/api/v1/repos/%s/git/notes/65f1bf27bc3bf70f64657658635e66094edbcb4d", repo.FullName())
		resp = MakeRequest(t, req, http.StatusOK)
		DecodeJSON(t, resp, &apiData)
		assert.Equal(t, "This is a new note\n", apiData.Message)
	})
}

func TestAPIReposDeleteGitNotes(t *testing.T) {
	onGiteaRun(t, func(*testing.T, *url.URL) {
		repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
		user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})

		session := loginUser(t, user.Name)
		token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository)

		req := NewRequestf(t, "GET", "/api/v1/repos/%s/git/notes/65f1bf27bc3bf70f64657658635e66094edbcb4d", repo.FullName())
		resp := MakeRequest(t, req, http.StatusOK)
		var apiData api.Note
		DecodeJSON(t, resp, &apiData)
		assert.Equal(t, "This is a test note\n", apiData.Message)

		req = NewRequestf(t, "DELETE", "/api/v1/repos/%s/git/notes/65f1bf27bc3bf70f64657658635e66094edbcb4d", repo.FullName()).AddTokenAuth(token)
		MakeRequest(t, req, http.StatusNoContent)

		req = NewRequestf(t, "GET", "/api/v1/repos/%s/git/notes/65f1bf27bc3bf70f64657658635e66094edbcb4d", repo.FullName())
		MakeRequest(t, req, http.StatusNotFound)
	})
}
