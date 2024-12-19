// Copyright 2024 The Forgejo Authors c/o Codeberg e.V.. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"testing"

	actions_model "code.gitea.io/gitea/models/actions"
	auth_model "code.gitea.io/gitea/models/auth"
	"code.gitea.io/gitea/models/unittest"
	"code.gitea.io/gitea/routers/api/v1/shared"
	"code.gitea.io/gitea/tests"
	"github.com/stretchr/testify/assert"
)

func TestAPISearchActionJobs_GlobalRunner(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	job := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRunJob{ID: 393})
	adminUsername := "user1"
	token := getUserToken(t, adminUsername, auth_model.AccessTokenScopeWriteAdmin)

	req := NewRequest(
		t,
		"GET",
		fmt.Sprintf("/api/v1/admin/runners/jobs?labels=%s", "ubuntu-latest"),
	).AddTokenAuth(token)
	res := MakeRequest(t, req, http.StatusOK)

	var jobs shared.RunJobList
	DecodeJSON(t, res, &jobs)

	assert.EqualValues(t, 1, len(jobs.Body))
	assert.EqualValues(t, job.ID, jobs.Body[0].ID)
}
