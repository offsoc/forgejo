// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"testing"

	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	api "forgejo.org/modules/structs"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
)

func TestAPIRepoPulls(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// repo = user2/repo1
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})

	// issue id without assigned review member or review team
	issueID := 5
	req := NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/pulls/%d", repo.OwnerName, repo.Name, issueID))
	res := MakeRequest(t, req, http.StatusOK)
	var pr *api.PullRequest
	DecodeJSON(t, res, &pr)

	assert.NotNil(t, pr.RequestedReviewers)
	assert.NotNil(t, pr.RequestedReviewersTeams)
}
