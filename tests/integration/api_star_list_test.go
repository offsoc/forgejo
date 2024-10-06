// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	auth_model "code.gitea.io/gitea/models/auth"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	api "code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIGetStarLists(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	require.NoError(t, unittest.LoadFixtures())

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})

	session := loginUser(t, user.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeReadUser, auth_model.AccessTokenScopeReadRepository)

	t.Run("CurrentUser", func(t *testing.T) {
		resp := MakeRequest(t, NewRequest(t, "GET", "/api/v1/user/starlists").AddTokenAuth(token), http.StatusOK)

		var starLists []api.StarList
		DecodeJSON(t, resp, &starLists)

		assert.Len(t, starLists, 2)

		assert.Equal(t, int64(1), starLists[0].ID)
		assert.Equal(t, "First List", starLists[0].Name)
		assert.Equal(t, "Description for first List", starLists[0].Description)
		assert.False(t, starLists[0].IsPrivate)

		assert.Equal(t, int64(2), starLists[1].ID)
		assert.Equal(t, "Second List", starLists[1].Name)
		assert.Equal(t, "This is private", starLists[1].Description)
		assert.True(t, starLists[1].IsPrivate)
	})

	t.Run("OtherUser", func(t *testing.T) {
		resp := MakeRequest(t, NewRequest(t, "GET", fmt.Sprintf("/api/v1/users/%s/starlists", user.Name)), http.StatusOK)

		var starLists []api.StarList
		DecodeJSON(t, resp, &starLists)

		assert.Len(t, starLists, 1)

		assert.Equal(t, int64(1), starLists[0].ID)
		assert.Equal(t, "First List", starLists[0].Name)
		assert.Equal(t, "Description for first List", starLists[0].Description)
		assert.False(t, starLists[0].IsPrivate)
	})
}

func TestAPIGetStarListRepoInfo(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	require.NoError(t, unittest.LoadFixtures())

	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})

	session := loginUser(t, user.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeReadUser, auth_model.AccessTokenScopeReadRepository)

	urlStr := fmt.Sprintf("/api/v1/user/starlists/repoinfo/%s/%s", repo.OwnerName, repo.Name)

	resp := MakeRequest(t, NewRequest(t, "GET", urlStr).AddTokenAuth(token), http.StatusOK)

	var repoInfo []api.StarListRepoInfo
	DecodeJSON(t, resp, &repoInfo)

	assert.Len(t, repoInfo, 2)

	assert.True(t, repoInfo[0].Contains)
	assert.Equal(t, int64(1), repoInfo[0].StarList.ID)

	assert.False(t, repoInfo[1].Contains)
	assert.Equal(t, int64(2), repoInfo[1].StarList.ID)
}

func TestAPICreateStarList(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	require.NoError(t, unittest.LoadFixtures())

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})

	session := loginUser(t, user.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteUser, auth_model.AccessTokenScopeWriteRepository)

	t.Run("Success", func(t *testing.T) {
		req := NewRequestWithJSON(t, "POST", "/api/v1/user/starlists", &api.CreateEditStarListOptions{
			Name:        "New Name",
			Description: "Hello",
			IsPrivate:   false,
		}).AddTokenAuth(token)
		resp := MakeRequest(t, req, http.StatusCreated)

		var starList api.StarList
		DecodeJSON(t, resp, &starList)

		assert.Equal(t, "New Name", starList.Name)
		assert.Equal(t, "Hello", starList.Description)
		assert.False(t, starList.IsPrivate)
	})

	t.Run("ExistingName", func(t *testing.T) {
		req := NewRequestWithJSON(t, "POST", "/api/v1/user/starlists", &api.CreateEditStarListOptions{
			Name:        "First List",
			Description: "Hello",
			IsPrivate:   false,
		}).AddTokenAuth(token)
		MakeRequest(t, req, http.StatusBadRequest)
	})
}

func TestAPIGetStarListByName(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	require.NoError(t, unittest.LoadFixtures())

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})

	session := loginUser(t, user.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeReadUser, auth_model.AccessTokenScopeReadRepository)

	t.Run("CurrentUserPublic", func(t *testing.T) {
		resp := MakeRequest(t, NewRequest(t, "GET", fmt.Sprintf("/api/v1/user/starlist/%s", url.PathEscape("First List"))).AddTokenAuth(token), http.StatusOK)

		var starList api.StarList
		DecodeJSON(t, resp, &starList)

		assert.Equal(t, int64(1), starList.ID)
		assert.Equal(t, "First List", starList.Name)
		assert.Equal(t, "Description for first List", starList.Description)
		assert.False(t, starList.IsPrivate)
	})

	t.Run("OtherUserPublic", func(t *testing.T) {
		resp := MakeRequest(t, NewRequest(t, "GET", fmt.Sprintf("/api/v1/users/%s/starlist/%s", user.Name, url.PathEscape("First List"))), http.StatusOK)

		var starList api.StarList
		DecodeJSON(t, resp, &starList)

		assert.Equal(t, int64(1), starList.ID)
		assert.Equal(t, "First List", starList.Name)
		assert.Equal(t, "Description for first List", starList.Description)
		assert.False(t, starList.IsPrivate)
	})

	t.Run("CurrentUserPrivate", func(t *testing.T) {
		resp := MakeRequest(t, NewRequest(t, "GET", fmt.Sprintf("/api/v1/user/starlist/%s", url.PathEscape("Second List"))).AddTokenAuth(token), http.StatusOK)

		var starList api.StarList
		DecodeJSON(t, resp, &starList)

		assert.Equal(t, int64(2), starList.ID)
		assert.Equal(t, "Second List", starList.Name)
		assert.Equal(t, "This is private", starList.Description)
		assert.True(t, starList.IsPrivate)
	})

	t.Run("OtherUserPublic", func(t *testing.T) {
		MakeRequest(t, NewRequest(t, "GET", fmt.Sprintf("/api/v1/users/%s/starlist/%s", user.Name, url.PathEscape("Second List"))), http.StatusNotFound)
	})
}

func TestAPIEditStarList(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	require.NoError(t, unittest.LoadFixtures())

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})

	session := loginUser(t, user.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteUser, auth_model.AccessTokenScopeWriteRepository)

	req := NewRequestWithJSON(t, "PATCH", fmt.Sprintf("/api/v1/user/starlist/%s", url.PathEscape("First List")), &api.CreateEditStarListOptions{
		Name:        "New Name",
		Description: "Hello",
		IsPrivate:   false,
	}).AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusOK)

	var starList api.StarList
	DecodeJSON(t, resp, &starList)

	assert.Equal(t, "New Name", starList.Name)
	assert.Equal(t, "Hello", starList.Description)
	assert.False(t, starList.IsPrivate)
}

func TestAPIDeleteStarList(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	require.NoError(t, unittest.LoadFixtures())

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})

	session := loginUser(t, user.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteUser, auth_model.AccessTokenScopeWriteRepository)

	MakeRequest(t, NewRequest(t, "DELETE", fmt.Sprintf("/api/v1/user/starlist/%s", url.PathEscape("First List"))).AddTokenAuth(token), http.StatusNoContent)
}

func TestAPIStarListGetRepos(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	require.NoError(t, unittest.LoadFixtures())

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})

	session := loginUser(t, user.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeReadUser, auth_model.AccessTokenScopeReadRepository)

	t.Run("CurrentUser", func(t *testing.T) {
		urlStr := fmt.Sprintf("/api/v1/user/starlist/%s/repos", url.PathEscape("First List"))
		resp := MakeRequest(t, NewRequest(t, "GET", urlStr).AddTokenAuth(token), http.StatusOK)

		var repoList []*api.Repository
		DecodeJSON(t, resp, &repoList)

		assert.Len(t, repoList, 1)

		assert.Equal(t, int64(1), repoList[0].ID)
	})

	t.Run("OtherUser", func(t *testing.T) {
		urlStr := fmt.Sprintf("/api/v1/users/%s/starlist/%s/repos", user.Name, url.PathEscape("First List"))
		resp := MakeRequest(t, NewRequest(t, "GET", urlStr), http.StatusOK)

		var repoList []*api.Repository
		DecodeJSON(t, resp, &repoList)

		assert.Len(t, repoList, 1)

		assert.Equal(t, int64(1), repoList[0].ID)
	})
}

func TestAPIStarListAddRepo(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	require.NoError(t, unittest.LoadFixtures())

	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 4})
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})

	session := loginUser(t, user.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteUser, auth_model.AccessTokenScopeWriteRepository)

	urlStr := fmt.Sprintf("/api/v1/user/starlist/%s/%s/%s", url.PathEscape("First List"), repo.OwnerName, repo.Name)
	MakeRequest(t, NewRequest(t, "PUT", urlStr).AddTokenAuth(token), http.StatusCreated)
}

func TestAPIStarListRemoveRepo(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	require.NoError(t, unittest.LoadFixtures())

	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})

	session := loginUser(t, user.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteUser, auth_model.AccessTokenScopeWriteRepository)

	urlStr := fmt.Sprintf("/api/v1/user/starlist/%s/%s/%s", url.PathEscape("First List"), repo.OwnerName, repo.Name)
	MakeRequest(t, NewRequest(t, "DELETE", urlStr).AddTokenAuth(token), http.StatusNoContent)
}

func TestAPIGetStarListByID(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	require.NoError(t, unittest.LoadFixtures())

	t.Run("PublicList", func(t *testing.T) {
		resp := MakeRequest(t, NewRequest(t, "GET", "/api/v1/starlist/1"), http.StatusOK)

		var starList api.StarList
		DecodeJSON(t, resp, &starList)

		assert.Equal(t, int64(1), starList.ID)
		assert.Equal(t, "First List", starList.Name)
		assert.Equal(t, "Description for first List", starList.Description)
		assert.False(t, starList.IsPrivate)
	})

	t.Run("PrivateList", func(t *testing.T) {
		MakeRequest(t, NewRequest(t, "GET", "/api/v1/starlist/2"), http.StatusNotFound)
	})
}
