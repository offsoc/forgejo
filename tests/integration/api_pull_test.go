// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	auth_model "forgejo.org/models/auth"
	"forgejo.org/models/db"
	issues_model "forgejo.org/models/issues"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/setting"
	api "forgejo.org/modules/structs"
	"forgejo.org/modules/test"
	"forgejo.org/services/forms"
	issue_service "forgejo.org/services/issue"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIViewPulls(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})

	ctx := NewAPITestContext(t, "user2", repo.Name, auth_model.AccessTokenScopeReadRepository)

	req := NewRequestf(t, "GET", "/api/v1/repos/%s/%s/pulls?state=all", owner.Name, repo.Name).
		AddTokenAuth(ctx.Token)
	resp := ctx.Session.MakeRequest(t, req, http.StatusOK)

	var pulls []*api.PullRequest
	DecodeJSON(t, resp, &pulls)
	expectedLen := unittest.GetCount(t, &issues_model.Issue{RepoID: repo.ID}, unittest.Cond("is_pull = ?", true))
	assert.Len(t, pulls, expectedLen)

	pull := pulls[0]
	if assert.EqualValues(t, 5, pull.ID) {
		resp = ctx.Session.MakeRequest(t, NewRequest(t, "GET", pull.DiffURL), http.StatusOK)
		_, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		// TODO: use diff to generate stats to test against

		t.Run(fmt.Sprintf("APIGetPullFiles_%d", pull.ID),
			doAPIGetPullFiles(ctx, pull, func(t *testing.T, files []*api.ChangedFile) {
				if assert.Len(t, files, 1) {
					assert.Equal(t, "File-WoW", files[0].Filename)
					assert.Empty(t, files[0].PreviousFilename)
					assert.Equal(t, 1, files[0].Additions)
					assert.Equal(t, 1, files[0].Changes)
					assert.Equal(t, 0, files[0].Deletions)
					assert.Equal(t, "added", files[0].Status)
				}
			}))
	}
}

func TestAPIViewPullsByBaseHead(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})

	ctx := NewAPITestContext(t, "user2", repo.Name, auth_model.AccessTokenScopeReadRepository)

	req := NewRequestf(t, "GET", "/api/v1/repos/%s/%s/pulls/master/branch2", owner.Name, repo.Name).
		AddTokenAuth(ctx.Token)
	resp := ctx.Session.MakeRequest(t, req, http.StatusOK)

	pull := &api.PullRequest{}
	DecodeJSON(t, resp, pull)
	assert.EqualValues(t, 3, pull.Index)
	assert.EqualValues(t, 2, pull.ID)

	req = NewRequestf(t, "GET", "/api/v1/repos/%s/%s/pulls/master/branch-not-exist", owner.Name, repo.Name).
		AddTokenAuth(ctx.Token)
	ctx.Session.MakeRequest(t, req, http.StatusNotFound)
}

// TestAPIMergePullWIP ensures that we can't merge a WIP pull request
func TestAPIMergePullWIP(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})
	pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{Status: issues_model.PullRequestStatusMergeable}, unittest.Cond("has_merged = ?", false))
	pr.LoadIssue(db.DefaultContext)
	issue_service.ChangeTitle(db.DefaultContext, pr.Issue, owner, setting.Repository.PullRequest.WorkInProgressPrefixes[0]+" "+pr.Issue.Title)

	// force reload
	pr.LoadAttributes(db.DefaultContext)

	assert.Contains(t, pr.Issue.Title, setting.Repository.PullRequest.WorkInProgressPrefixes[0])

	session := loginUser(t, owner.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository)
	req := NewRequestWithJSON(t, http.MethodPost, fmt.Sprintf("/api/v1/repos/%s/%s/pulls/%d/merge", owner.Name, repo.Name, pr.Index), &forms.MergePullRequestForm{
		MergeMessageField: pr.Issue.Title,
		Do:                string(repo_model.MergeStyleMerge),
	}).AddTokenAuth(token)

	MakeRequest(t, req, http.StatusMethodNotAllowed)
}

func TestAPICreatePullSuccess(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	repo10 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 10})
	// repo10 have code, pulls units.
	repo11 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 11})
	// repo11 only have code unit but should still create pulls
	owner10 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo10.OwnerID})
	owner11 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo11.OwnerID})

	session := loginUser(t, owner11.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository)
	req := NewRequestWithJSON(t, http.MethodPost, fmt.Sprintf("/api/v1/repos/%s/%s/pulls", owner10.Name, repo10.Name), &api.CreatePullRequestOption{
		Head:  fmt.Sprintf("%s:master", owner11.Name),
		Base:  "master",
		Title: "create a failure pr",
	}).AddTokenAuth(token)
	MakeRequest(t, req, http.StatusCreated)
	MakeRequest(t, req, http.StatusUnprocessableEntity) // second request should fail
}

func TestAPICreatePullSameRepoSuccess(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})

	session := loginUser(t, owner.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository)

	req := NewRequestWithJSON(t, http.MethodPost, fmt.Sprintf("/api/v1/repos/%s/%s/pulls", owner.Name, repo.Name), &api.CreatePullRequestOption{
		Head:  fmt.Sprintf("%s:pr-to-update", owner.Name),
		Base:  "master",
		Title: "successfully create a PR between branches of the same repository",
	}).AddTokenAuth(token)
	MakeRequest(t, req, http.StatusCreated)
	MakeRequest(t, req, http.StatusUnprocessableEntity) // second request should fail
}

func TestAPICreatePullWithFieldsSuccess(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	// repo10 have code, pulls units.
	repo10 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 10})
	owner10 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo10.OwnerID})
	// repo11 only have code unit but should still create pulls
	repo11 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 11})
	owner11 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo11.OwnerID})

	session := loginUser(t, owner11.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository)

	opts := &api.CreatePullRequestOption{
		Head:      fmt.Sprintf("%s:master", owner11.Name),
		Base:      "master",
		Title:     "create a failure pr",
		Body:      "foobaaar",
		Milestone: 5,
		Assignees: []string{owner10.Name},
		Labels:    []int64{5},
	}

	req := NewRequestWithJSON(t, http.MethodPost, fmt.Sprintf("/api/v1/repos/%s/%s/pulls", owner10.Name, repo10.Name), opts).
		AddTokenAuth(token)

	res := MakeRequest(t, req, http.StatusCreated)
	pull := new(api.PullRequest)
	DecodeJSON(t, res, pull)

	assert.NotNil(t, pull.Milestone)
	assert.Equal(t, opts.Milestone, pull.Milestone.ID)
	if assert.Len(t, pull.Assignees, 1) {
		assert.Equal(t, opts.Assignees[0], owner10.Name)
	}
	assert.NotNil(t, pull.Labels)
	assert.Equal(t, opts.Labels[0], pull.Labels[0].ID)
}

func TestAPICreatePullWithFieldsFailure(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	// repo10 have code, pulls units.
	repo10 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 10})
	owner10 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo10.OwnerID})
	// repo11 only have code unit but should still create pulls
	repo11 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 11})
	owner11 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo11.OwnerID})

	session := loginUser(t, owner11.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository)

	opts := &api.CreatePullRequestOption{
		Head: fmt.Sprintf("%s:master", owner11.Name),
		Base: "master",
	}

	req := NewRequestWithJSON(t, http.MethodPost, fmt.Sprintf("/api/v1/repos/%s/%s/pulls", owner10.Name, repo10.Name), opts).
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusUnprocessableEntity)
	opts.Title = "is required"

	opts.Milestone = 666
	MakeRequest(t, req, http.StatusUnprocessableEntity)
	opts.Milestone = 5

	opts.Assignees = []string{"qweruqweroiuyqweoiruywqer"}
	MakeRequest(t, req, http.StatusUnprocessableEntity)
	opts.Assignees = []string{owner10.LoginName}

	opts.Labels = []int64{55555}
	MakeRequest(t, req, http.StatusUnprocessableEntity)
	opts.Labels = []int64{5}
}

func TestAPIEditPull(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	repo10 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 10})
	owner10 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo10.OwnerID})

	session := loginUser(t, owner10.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository)
	title := "create a success pr"
	req := NewRequestWithJSON(t, http.MethodPost, fmt.Sprintf("/api/v1/repos/%s/%s/pulls", owner10.Name, repo10.Name), &api.CreatePullRequestOption{
		Head:  "develop",
		Base:  "master",
		Title: title,
	}).AddTokenAuth(token)
	apiPull := new(api.PullRequest)
	resp := MakeRequest(t, req, http.StatusCreated)
	DecodeJSON(t, resp, apiPull)
	assert.Equal(t, "master", apiPull.Base.Name)

	newTitle := "edit a this pr"
	newBody := "edited body"
	urlStr := fmt.Sprintf("/api/v1/repos/%s/%s/pulls/%d", owner10.Name, repo10.Name, apiPull.Index)
	req = NewRequestWithJSON(t, http.MethodPatch, urlStr, &api.EditPullRequestOption{
		Base:  "feature/1",
		Title: newTitle,
		Body:  &newBody,
	}).AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusCreated)
	DecodeJSON(t, resp, apiPull)
	assert.Equal(t, "feature/1", apiPull.Base.Name)
	// check comment history
	pull := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: apiPull.ID})
	err := pull.LoadIssue(db.DefaultContext)
	require.NoError(t, err)
	unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{IssueID: pull.Issue.ID, OldTitle: title, NewTitle: newTitle})
	unittest.AssertExistsAndLoadBean(t, &issues_model.ContentHistory{IssueID: pull.Issue.ID, ContentText: newBody, IsFirstCreated: false})

	// verify the idempotency of a state change
	pullState := string(apiPull.State)
	req = NewRequestWithJSON(t, http.MethodPatch, urlStr, &api.EditPullRequestOption{
		State: &pullState,
	}).AddTokenAuth(token)
	apiPullIdempotent := new(api.PullRequest)
	resp = MakeRequest(t, req, http.StatusCreated)
	DecodeJSON(t, resp, apiPullIdempotent)
	assert.Equal(t, apiPull.State, apiPullIdempotent.State)

	req = NewRequestWithJSON(t, http.MethodPatch, urlStr, &api.EditPullRequestOption{
		Base: "not-exist",
	}).AddTokenAuth(token)
	MakeRequest(t, req, http.StatusNotFound)
}

func TestAPIForkDifferentName(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// Step 1: get a repo and a user that can fork this repo
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 5})

	session := loginUser(t, user.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)

	// Step 2: fork this repo with another name
	forkName := "myfork"
	req := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/forks", owner.Name, repo.Name),
		&api.CreateForkOption{Name: &forkName}).AddTokenAuth(token)
	MakeRequest(t, req, http.StatusAccepted)

	// Step 3: make a PR onto the original repo, it should succeed
	req = NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/pulls?state=all", owner.Name, repo.Name),
		&api.CreatePullRequestOption{Head: user.Name + ":master", Base: "master", Title: "hi"}).AddTokenAuth(token)
	MakeRequest(t, req, http.StatusCreated)
}

func doAPIGetPullFiles(ctx APITestContext, pr *api.PullRequest, callback func(*testing.T, []*api.ChangedFile)) func(*testing.T) {
	return func(t *testing.T) {
		req := NewRequest(t, http.MethodGet, fmt.Sprintf("/api/v1/repos/%s/%s/pulls/%d/files", ctx.Username, ctx.Reponame, pr.Index)).
			AddTokenAuth(ctx.Token)
		if ctx.ExpectedCode == 0 {
			ctx.ExpectedCode = http.StatusOK
		}
		resp := ctx.Session.MakeRequest(t, req, ctx.ExpectedCode)

		files := make([]*api.ChangedFile, 0, 1)
		DecodeJSON(t, resp, &files)

		if callback != nil {
			callback(t, files)
		}
	}
}

func TestAPIPullDeleteBranchPerms(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, giteaURL *url.URL) {
		user2Session := loginUser(t, "user2")
		user4Session := loginUser(t, "user4")
		testRepoFork(t, user4Session, "user2", "repo1", "user4", "repo1")
		testEditFileToNewBranch(t, user2Session, "user2", "repo1", "master", "base-pr", "README.md", "Hello, World\n(Edited - base PR)\n")

		req := NewRequestWithValues(t, "POST", "/user4/repo1/compare/master...user2/repo1:base-pr", map[string]string{
			"_csrf": GetCSRF(t, user4Session, "/user4/repo1/compare/master...user2/repo1:base-pr"),
			"title": "Testing PR",
		})
		resp := user4Session.MakeRequest(t, req, http.StatusOK)
		elem := strings.Split(test.RedirectURL(resp), "/")

		token := getTokenForLoggedInUser(t, user4Session, auth_model.AccessTokenScopeWriteRepository)
		req = NewRequestWithValues(t, "POST", "/api/v1/repos/user4/repo1/pulls/"+elem[4]+"/merge", map[string]string{
			"do":                        "merge",
			"delete_branch_after_merge": "on",
		}).AddTokenAuth(token)
		resp = user4Session.MakeRequest(t, req, http.StatusForbidden)

		type userResponse struct {
			Message string `json:"message"`
		}
		var bodyResp userResponse
		DecodeJSON(t, resp, &bodyResp)

		assert.Equal(t, "insufficient permission to delete head branch", bodyResp.Message)

		// Check that the branch still exist.
		req = NewRequest(t, "GET", "/api/v1/repos/user2/repo1/branches/base-pr").AddTokenAuth(token)
		user4Session.MakeRequest(t, req, http.StatusOK)
	})
}
