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
	api "code.gitea.io/gitea/modules/structs"
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

	var jobs []*api.ActionRunJob
	DecodeJSON(t, res, &jobs)

	assert.Len(t, jobs, 1)
	assert.EqualValues(t, job.ID, jobs[0].ID)
}
