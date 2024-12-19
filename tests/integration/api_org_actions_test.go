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

func TestAPISearchActionJobs_OrgRunner(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	session := loginUser(t, "user1")
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteOrganization)

	job := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRunJob{ID: 395})

	req := NewRequest(t, "GET",
		fmt.Sprintf("/api/v1/orgs/org3/actions/runners/jobs?labels=%s", "fedora")).
		AddTokenAuth(token)
	res := MakeRequest(t, req, http.StatusOK)

	var jobs shared.RunJobList
	DecodeJSON(t, res, &jobs)

	assert.Len(t, len(jobs.Body), 1)
	assert.EqualValues(t, job.ID, jobs.Body[0].ID)
}
