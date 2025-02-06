// Copyright 2021 The Gitea Authors. All rights reserved.
// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"testing"

	auth_model "code.gitea.io/gitea/models/auth"
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/perm"
	unit_model "code.gitea.io/gitea/models/unit"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitSmartHTTP(t *testing.T) {
	onGiteaRun(t, testGitSmartHTTP)
}

func testGitSmartHTTP(t *testing.T, u *url.URL) {
	kases := []struct {
		p    string
		code int
	}{
		{
			p:    "user2/repo1/info/refs",
			code: http.StatusOK,
		},
		{
			p:    "user2/repo1/HEAD",
			code: http.StatusOK,
		},
		{
			p:    "user2/repo1/objects/info/alternates",
			code: http.StatusNotFound,
		},
		{
			p:    "user2/repo1/objects/info/http-alternates",
			code: http.StatusNotFound,
		},
		{
			p:    "user2/repo1/../../custom/conf/app.ini",
			code: http.StatusNotFound,
		},
		{
			p:    "user2/repo1/objects/info/../../../../custom/conf/app.ini",
			code: http.StatusNotFound,
		},
		{
			p:    `user2/repo1/objects/info/..\..\..\..\custom\conf\app.ini`,
			code: http.StatusBadRequest,
		},
	}

	for _, kase := range kases {
		t.Run(kase.p, func(t *testing.T) {
			p := u.String() + kase.p
			req, err := http.NewRequest("GET", p, nil)
			require.NoError(t, err)
			req.SetBasicAuth("user2", userPassword)
			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.EqualValues(t, kase.code, resp.StatusCode)
			_, err = io.ReadAll(resp.Body)
			require.NoError(t, err)
		})
	}
}

// Test that the git http endpoints have the same authentication behavior irrespective of if it is a GET or a HEAD request.
func TestGitHTTPSameStatusCodeForGetAndHeadRequests(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
	user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	type caseType struct {
		User               *user_model.User
		IsCollaborator     bool
		RepoIsPrivate      bool
		Endpoint           string
		ExpectedStatusCode int
	}
	cases := []caseType{
		{owner, false, false, "HEAD", 200},
		{owner, false, false, "git-receive-pack", 405},
		{owner, false, false, "git-upload-pack", 405},
		{owner, false, false, "info/refs", 200},
		{owner, false, false, "objects/info/alternates", 404},
		{owner, false, false, "objects/info/http-alternates", 404},
		{owner, false, false, "objects/info/packs", 200},
		{owner, false, true, "HEAD", 200},
		{owner, false, true, "git-receive-pack", 405},
		{owner, false, true, "git-upload-pack", 405},
		{owner, false, true, "info/refs", 200},
		{owner, false, true, "objects/info/alternates", 404},
		{owner, false, true, "objects/info/http-alternates", 404},
		{owner, false, true, "objects/info/packs", 200},
		{user2, false, false, "HEAD", 200},
		{user2, false, false, "git-receive-pack", 405},
		{user2, false, false, "git-upload-pack", 405},
		{user2, false, false, "info/refs", 200},
		{user2, false, false, "objects/info/alternates", 404},
		{user2, false, false, "objects/info/http-alternates", 404},
		{user2, false, false, "objects/info/packs", 200},
		{user2, false, true, "HEAD", 404},
		{user2, false, true, "git-receive-pack", 405},
		{user2, false, true, "git-upload-pack", 405},
		{user2, false, true, "info/refs", 404},
		{user2, false, true, "objects/info/alternates", 404},
		{user2, false, true, "objects/info/http-alternates", 404},
		{user2, false, true, "objects/info/packs", 404},
		// user2 with IsCollaborator=true must come after IsCollaborator=false, because
		// the addition as a collaborator is not reset
		{user2, true, false, "HEAD", 200},
		{user2, true, false, "git-receive-pack", 405},
		{user2, true, false, "git-upload-pack", 405},
		{user2, true, false, "info/refs", 200},
		{user2, true, false, "objects/info/alternates", 404},
		{user2, true, false, "objects/info/http-alternates", 404},
		{user2, true, false, "objects/info/packs", 200},
		{user2, true, true, "HEAD", 200},
		{user2, true, true, "git-receive-pack", 405},
		{user2, true, true, "git-upload-pack", 405},
		{user2, true, true, "info/refs", 200},
		{user2, true, true, "objects/info/alternates", 404},
		{user2, true, true, "objects/info/http-alternates", 404},
		{user2, true, true, "objects/info/packs", 200},
		{nil, false, false, "HEAD", 200},
		{nil, false, false, "git-receive-pack", 405},
		{nil, false, false, "git-upload-pack", 405},
		{nil, false, false, "info/refs", 200},
		{nil, false, false, "objects/info/alternates", 404},
		{nil, false, false, "objects/info/http-alternates", 404},
		{nil, false, false, "objects/info/packs", 200},
		{nil, false, true, "HEAD", 401},
		{nil, false, true, "git-receive-pack", 405},
		{nil, false, true, "git-upload-pack", 405},
		{nil, false, true, "info/refs", 401},
		{nil, false, true, "objects/info/alternates", 401},
		{nil, false, true, "objects/info/http-alternates", 401},
		{nil, false, true, "objects/info/packs", 401},
	}

	caseToTestName := func(c caseType) string {
		var user string
		if c.User == nil {
			user = "nil"
		} else if c.User == owner {
			user = "owner"
		} else {
			user = c.User.Name
		}
		return fmt.Sprintf(
			"User=%s,IsCollaborator=%t,RepoIsPrivate=%t,Endpoint=%s,ExpectedStatusCode=%d",
			user,
			c.IsCollaborator,
			c.RepoIsPrivate,
			c.Endpoint,
			c.ExpectedStatusCode,
		)
	}

	repo, _, f := tests.CreateDeclarativeRepo(t, owner, "get-and-head-requests", []unit_model.Type{unit_model.TypeCode}, nil, nil)
	defer f()

	for _, c := range cases {
		t.Run(caseToTestName(c), func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			session := emptyTestSession(t)
			if c.User != nil {
				session = loginUser(t, c.User.Name)
			}
			if c.IsCollaborator {
				testCtx := NewAPITestContext(t, owner.Name, repo.Name, auth_model.AccessTokenScopeWriteRepository)
				doAPIAddCollaborator(testCtx, c.User.Name, perm.AccessModeRead)(t)
			}
			repo.IsPrivate = c.RepoIsPrivate
			_, err := db.GetEngine(db.DefaultContext).Cols("is_private").Update(repo)
			require.NoError(t, err)

			// Given the test parameters check that the endpoint returns the same status
			// code for both GET and HEAD, which needs to equal the test cases expected
			// status code
			getReq := NewRequestf(t, "GET", "%s/%s", repo.Link(), c.Endpoint)
			getResp := session.MakeRequest(t, getReq, NoExpectedStatus)
			headReq := NewRequestf(t, "HEAD", "%s/%s", repo.Link(), c.Endpoint)
			headResp := session.MakeRequest(t, headReq, NoExpectedStatus)
			require.Equal(t, getResp.Result().StatusCode, headResp.Result().StatusCode)
			require.Equal(t, c.ExpectedStatusCode, headResp.Result().StatusCode)
		})
	}
}
