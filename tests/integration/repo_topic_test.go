// Copyright 2022 The Gitea Authors. All rights reserved.
// Copyright 2024 The Forgejo Authors c/o Codeberg e.V.. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"net/url"
	"testing"

	auth_model "forgejo.org/models/auth"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	api "forgejo.org/modules/structs"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
)

func TestTopicSearch(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	searchURL, _ := url.Parse("/explore/topics/search")
	var topics struct {
		TopicNames []*api.TopicResponse `json:"topics"`
	}

	query := url.Values{"page": []string{"1"}, "limit": []string{"4"}}

	searchURL.RawQuery = query.Encode()
	res := MakeRequest(t, NewRequest(t, "GET", searchURL.String()), http.StatusOK)
	DecodeJSON(t, res, &topics)
	assert.Len(t, topics.TopicNames, 4)
	assert.Equal(t, "6", res.Header().Get("x-total-count"))

	query.Add("q", "topic")
	searchURL.RawQuery = query.Encode()
	res = MakeRequest(t, NewRequest(t, "GET", searchURL.String()), http.StatusOK)
	DecodeJSON(t, res, &topics)
	assert.Len(t, topics.TopicNames, 2)

	query.Set("q", "database")
	searchURL.RawQuery = query.Encode()
	res = MakeRequest(t, NewRequest(t, "GET", searchURL.String()), http.StatusOK)
	DecodeJSON(t, res, &topics)
	if assert.Len(t, topics.TopicNames, 1) {
		assert.EqualValues(t, 2, topics.TopicNames[0].ID)
		assert.Equal(t, "database", topics.TopicNames[0].Name)
		assert.Equal(t, 1, topics.TopicNames[0].RepoCount)
	}
}

func TestTopicSearchPaging(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	var topics struct {
		TopicNames []*api.TopicResponse `json:"topics"`
	}

	// Add 20 unique topics to user2/repo2, and 20 unique ones to user2/repo3
	user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	token2 := getUserToken(t, user2.Name, auth_model.AccessTokenScopeWriteRepository)
	repo2 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	repo3 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 2})
	for i := 0; i < 20; i++ {
		req := NewRequestf(t, "PUT", "/api/v1/repos/%s/%s/topics/paging-topic-%d", user2.Name, repo2.Name, i).
			AddTokenAuth(token2)
		MakeRequest(t, req, http.StatusNoContent)
		req = NewRequestf(t, "PUT", "/api/v1/repos/%s/%s/topics/paging-topic-%d", user2.Name, repo3.Name, i+30).
			AddTokenAuth(token2)
		MakeRequest(t, req, http.StatusNoContent)
	}

	res := MakeRequest(t, NewRequest(t, "GET", "/explore/topics/search"), http.StatusOK)
	DecodeJSON(t, res, &topics)
	assert.Len(t, topics.TopicNames, 30)

	res = MakeRequest(t, NewRequest(t, "GET", "/explore/topics/search?page=2"), http.StatusOK)
	DecodeJSON(t, res, &topics)
	assert.NotEmpty(t, topics.TopicNames)
}
