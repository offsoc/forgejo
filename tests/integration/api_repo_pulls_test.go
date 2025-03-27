// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/unittest"
	api "code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/tests"
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func TestAPIRepoPulls(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// repo = user2/repo1
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})

	// issue id without assigned review member or review team
	issueId := 5
	req := NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/pulls/%d", repo.OwnerName, repo.Name, issueId))
	res := MakeRequest(t, req, http.StatusOK)
	var pr *api.PullRequest
	DecodeJSON(t, res, &pr)

	assert.NotNil(t, pr.RequestedReviewers)
	assert.NotNil(t, pr.RequestedReviewersTeams)
}
