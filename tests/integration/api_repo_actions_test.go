// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	actions_model "forgejo.org/models/actions"
	auth_model "forgejo.org/models/auth"
	repo_model "forgejo.org/models/repo"
	unit_model "forgejo.org/models/unit"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	api "forgejo.org/modules/structs"
	files_service "forgejo.org/services/repository/files"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
)

func TestAPISearchActionJobs_RepoRunner(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	token := getUserToken(t, user2.LowerName, auth_model.AccessTokenScopeWriteRepository)
	job := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRunJob{ID: 393})

	req := NewRequestf(
		t,
		"GET",
		"/api/v1/repos/%s/%s/actions/runners/jobs?labels=%s",
		repo.OwnerName, repo.Name,
		"ubuntu-latest",
	).AddTokenAuth(token)
	res := MakeRequest(t, req, http.StatusOK)

	var jobs []*api.ActionRunJob
	DecodeJSON(t, res, &jobs)

	assert.Len(t, jobs, 1)
	assert.Equal(t, job.ID, jobs[0].ID)
}

func TestAPIWorkflowDispatchReturnInfo(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		workflowName := "dispatch.yml"
		user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
		token := getUserToken(t, user2.LowerName, auth_model.AccessTokenScopeWriteRepository)

		// create the repo
		repo, _, f := tests.CreateDeclarativeRepo(t, user2, "api-repo-workflow-dispatch",
			[]unit_model.Type{unit_model.TypeActions}, nil,
			[]*files_service.ChangeRepoFile{
				{
					Operation: "create",
					TreePath:  fmt.Sprintf(".forgejo/workflows/%s", workflowName),
					ContentReader: strings.NewReader(`name: WD
on: [workflow-dispatch]
jobs:
  t1:
    runs-on: docker
    steps:
      - run: echo "test 1"
  t2:
    runs-on: docker
    steps:
      - run: echo "test 2"
`,
					),
				},
			},
		)
		defer f()

		req := NewRequestWithJSON(
			t,
			http.MethodPost,
			fmt.Sprintf(
				"/api/v1/repos/%s/%s/actions/workflows/%s/dispatches",
				repo.OwnerName, repo.Name, workflowName,
			),
			&api.DispatchWorkflowOption{
				Ref:           repo.DefaultBranch,
				ReturnRunInfo: true,
			},
		)
		req.AddTokenAuth(token)

		res := MakeRequest(t, req, http.StatusCreated)
		run := new(api.DispatchWorkflowRun)
		DecodeJSON(t, res, run)

		assert.NotZero(t, run.ID)
		assert.NotZero(t, run.RunNumber)
		assert.Len(t, run.Jobs, 2)
	})
}
