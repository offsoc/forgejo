package integration

import (
	actions_model "code.gitea.io/gitea/models/actions"
	auth_model "code.gitea.io/gitea/models/auth"
	"code.gitea.io/gitea/models/unittest"
	"code.gitea.io/gitea/routers/api/v1/shared"
	"code.gitea.io/gitea/tests"
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func TestAPISearchActionJobs_UserRunner(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	normalUsername := "user2"
	session := loginUser(t, normalUsername)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteUser)
	job := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRunJob{ID: 394})

	req := NewRequest(t, "GET",
		fmt.Sprintf("/api/v1/user/actions/runners/jobs?labels=%s", "debian-latest")).
		AddTokenAuth(token)
	res := MakeRequest(t, req, http.StatusOK)

	var jobs shared.RunJobList
	DecodeJSON(t, res, &jobs)

	assert.EqualValues(t, 1, len(jobs.Body))
	assert.EqualValues(t, job.ID, jobs.Body[0].ID)
}
