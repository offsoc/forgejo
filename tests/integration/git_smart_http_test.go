// Copyright 2021 The Gitea Authors. All rights reserved.
// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"io"
	"net/http"
	"net/url"
	"strconv"
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

	users := []struct {
		Name string
		User *user_model.User
	}{
		{Name: "Owner", User: owner},
		{Name: "User2", User: user2},
		{Name: "Anonymous", User: nil},
	}

	endpoints := []string{
		"HEAD",
		"git-receive-pack",
		"git-upload-pack",
		"info/refs",
		"objects/info/alternates",
		"objects/info/http-alternates",
		"objects/info/packs",
	}

	repo, _, f := tests.CreateDeclarativeRepo(t, owner, "get-and-head-requests", []unit_model.Type{unit_model.TypeCode}, nil, nil)
	defer f()

	for _, user := range users {
		t.Run("User="+user.Name, func(t *testing.T) {
			session := emptyTestSession(t)
			if user.User != nil {
				session = loginUser(t, user.User.Name)
			}
			for _, isCollaborator := range []bool{false, true} {
				// Adding the owner of the repository or anonymous as a collaborator makes no sense
				if (user.User == nil || user.User == owner) && isCollaborator {
					continue
				}
				t.Run("IsCollaborator="+strconv.FormatBool(isCollaborator), func(t *testing.T) {
					if isCollaborator {
						testCtx := NewAPITestContext(t, owner.Name, repo.Name, auth_model.AccessTokenScopeWriteRepository)
						doAPIAddCollaborator(testCtx, user.Name, perm.AccessModeRead)(t)
					}
					for _, repoIsPrivate := range []bool{false, true} {
						t.Run("repo.IsPrivate="+strconv.FormatBool(repoIsPrivate), func(t *testing.T) {
							repo.IsPrivate = repoIsPrivate
							_, err := db.GetEngine(db.DefaultContext).Cols("is_private").Update(repo)
							require.NoError(t, err)
							for _, endpoint := range endpoints {
								t.Run("Endpoint="+endpoint, func(t *testing.T) {
									defer tests.PrintCurrentTest(t)()
									// Given the other parameters check that the endpoint returns the same status
									// code for both GET and HEAD
									getReq := NewRequestf(t, "GET", "%s/%s", repo.Link(), endpoint)
									getResp := session.MakeRequest(t, getReq, NoExpectedStatus)
									headReq := NewRequestf(t, "HEAD", "%s/%s", repo.Link(), endpoint)
									headResp := session.MakeRequest(t, headReq, NoExpectedStatus)
									require.Equal(t, getResp.Result().StatusCode, headResp.Result().StatusCode)
									if user.User == nil && endpoint == "HEAD" {
										// Sanity check: anonymous requests for the HEAD endpoint should result in a 401
										// for private repositories and a 200 for public ones
										if repo.IsPrivate {
											require.Equal(t, 401, headResp.Result().StatusCode)
										} else {
											require.Equal(t, 200, headResp.Result().StatusCode)
										}
									}
								})
							}
						})
					}
				})
			}
		})
	}
}
