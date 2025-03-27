// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"testing"

	"forgejo.org/models/db"
	issues_model "forgejo.org/models/issues"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	api "forgejo.org/modules/structs"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIPullCommits(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 2})
	require.NoError(t, pr.LoadIssue(db.DefaultContext))
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: pr.HeadRepoID})

	req := NewRequestf(t, http.MethodGet, "/api/v1/repos/%s/%s/pulls/%d/commits", repo.OwnerName, repo.Name, pr.Index)
	resp := MakeRequest(t, req, http.StatusOK)

	var commits []*api.Commit
	DecodeJSON(t, resp, &commits)

	if !assert.Len(t, commits, 2) {
		return
	}

	assert.Equal(t, "5f22f7d0d95d614d25a5b68592adb345a4b5c7fd", commits[0].SHA)
	assert.Equal(t, "4a357436d925b5c974181ff12a994538ddc5a269", commits[1].SHA)

	assert.NotEmpty(t, commits[0].Files)
	assert.NotEmpty(t, commits[1].Files)
	assert.NotNil(t, commits[0].RepoCommit.Verification)
	assert.NotNil(t, commits[1].RepoCommit.Verification)
}

// TODO add tests for already merged PR and closed PR
