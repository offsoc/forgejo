// Copyright The Forgejo Authors
// SPDX-License-Identifier: MIT

package integration

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	auth_model "code.gitea.io/gitea/models/auth"
	"code.gitea.io/gitea/models/db"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/setting"
	api "code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/modules/test"
	"code.gitea.io/gitea/services/migrations"
	mirror_service "code.gitea.io/gitea/services/mirror"
	repo_service "code.gitea.io/gitea/services/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIPushMirror(t *testing.T) {
	onGiteaRun(t, testAPIPushMirror)
}

func testAPIPushMirror(t *testing.T, u *url.URL) {
	defer test.MockVariableValue(&setting.Migrations.AllowLocalNetworks, true)()
	defer test.MockVariableValue(&setting.Mirror.Enabled, true)()
	defer test.MockProtect(&mirror_service.AddPushMirrorRemote)()
	defer test.MockProtect(&repo_model.DeletePushMirrors)()

	require.NoError(t, migrations.Init())

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
	srcRepo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: srcRepo.OwnerID})
	session := loginUser(t, user.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeAll)
	urlStr := fmt.Sprintf("/api/v1/repos/%s/%s/push_mirrors", owner.Name, srcRepo.Name)

	mirrorRepo, err := repo_service.CreateRepositoryDirectly(db.DefaultContext, user, user, repo_service.CreateRepoOptions{
		Name: "test-push-mirror",
	})
	require.NoError(t, err)
	remoteAddress := fmt.Sprintf("%s%s/%s", u.String(), url.PathEscape(user.Name), url.PathEscape(mirrorRepo.Name))

	deletePushMirrors := repo_model.DeletePushMirrors
	deletePushMirrorsError := "deletePushMirrorsError"
	deletePushMirrorsFail := func(ctx context.Context, opts repo_model.PushMirrorOptions) error {
		return fmt.Errorf(deletePushMirrorsError)
	}

	addPushMirrorRemote := mirror_service.AddPushMirrorRemote
	addPushMirrorRemoteError := "addPushMirrorRemoteError"
	addPushMirrorRemoteFail := func(ctx context.Context, m *repo_model.PushMirror, addr string) error {
		return fmt.Errorf(addPushMirrorRemoteError)
	}

	for _, testCase := range []struct {
		name        string
		message     string
		status      int
		mirrorCount int
		setup       func()
	}{
		{
			name:        "success",
			status:      http.StatusOK,
			mirrorCount: 1,
			setup: func() {
				mirror_service.AddPushMirrorRemote = addPushMirrorRemote
				repo_model.DeletePushMirrors = deletePushMirrors
			},
		},
		{
			name:        "fail to add and delete",
			message:     deletePushMirrorsError,
			status:      http.StatusInternalServerError,
			mirrorCount: 1,
			setup: func() {
				mirror_service.AddPushMirrorRemote = addPushMirrorRemoteFail
				repo_model.DeletePushMirrors = deletePushMirrorsFail
			},
		},
		{
			name:        "fail to add",
			message:     addPushMirrorRemoteError,
			status:      http.StatusInternalServerError,
			mirrorCount: 0,
			setup: func() {
				mirror_service.AddPushMirrorRemote = addPushMirrorRemoteFail
				repo_model.DeletePushMirrors = deletePushMirrors
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.setup()
			req := NewRequestWithJSON(t, "POST", urlStr, &api.CreatePushMirrorOption{
				RemoteAddress: remoteAddress,
				Interval:      "8h",
			}).AddTokenAuth(token)

			resp := MakeRequest(t, req, testCase.status)
			if testCase.message != "" {
				err := api.APIError{}
				DecodeJSON(t, resp, &err)
				assert.EqualValues(t, testCase.message, err.Message)
			}

			req = NewRequest(t, "GET", urlStr).AddTokenAuth(token)
			resp = MakeRequest(t, req, http.StatusOK)
			var pushMirrors []*api.PushMirror
			DecodeJSON(t, resp, &pushMirrors)
			if assert.Len(t, pushMirrors, testCase.mirrorCount) && testCase.mirrorCount > 0 {
				pushMirror := pushMirrors[0]
				assert.EqualValues(t, remoteAddress, pushMirror.RemoteAddress)

				repo_model.DeletePushMirrors = deletePushMirrors
				req = NewRequest(t, "DELETE", fmt.Sprintf("%s/%s", urlStr, pushMirror.RemoteName)).AddTokenAuth(token)
				MakeRequest(t, req, http.StatusNoContent)
			}
		})
	}
}
