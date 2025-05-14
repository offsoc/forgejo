// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	actions_model "forgejo.org/models/actions"
	auth_model "forgejo.org/models/auth"
	issues_model "forgejo.org/models/issues"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	api "forgejo.org/modules/structs"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIModifyLabels(t *testing.T) {
	require.NoError(t, unittest.LoadFixtures())

	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 2})
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})
	session := loginUser(t, owner.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteIssue)
	urlStr := fmt.Sprintf("/api/v1/repos/%s/%s/labels", owner.Name, repo.Name)

	// CreateLabel
	req := NewRequestWithJSON(t, "POST", urlStr, &api.CreateLabelOption{
		Name:        "TestL 1",
		Color:       "abcdef",
		Description: "test label",
	}).AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusCreated)
	apiLabel := new(api.Label)
	DecodeJSON(t, resp, &apiLabel)
	dbLabel := unittest.AssertExistsAndLoadBean(t, &issues_model.Label{ID: apiLabel.ID, RepoID: repo.ID})
	assert.Equal(t, dbLabel.Name, apiLabel.Name)
	assert.Equal(t, strings.TrimLeft(dbLabel.Color, "#"), apiLabel.Color)

	req = NewRequestWithJSON(t, "POST", urlStr, &api.CreateLabelOption{
		Name:        "TestL 2",
		Color:       "#123456",
		Description: "jet another test label",
	}).AddTokenAuth(token)
	MakeRequest(t, req, http.StatusCreated)
	req = NewRequestWithJSON(t, "POST", urlStr, &api.CreateLabelOption{
		Name:  "WrongTestL",
		Color: "#12345g",
	}).AddTokenAuth(token)
	MakeRequest(t, req, http.StatusUnprocessableEntity)

	// ListLabels
	req = NewRequest(t, "GET", urlStr).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)
	var apiLabels []*api.Label
	DecodeJSON(t, resp, &apiLabels)
	assert.Len(t, apiLabels, 2)

	// GetLabel
	singleURLStr := fmt.Sprintf("/api/v1/repos/%s/%s/labels/%d", owner.Name, repo.Name, dbLabel.ID)
	req = NewRequest(t, "GET", singleURLStr).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)
	DecodeJSON(t, resp, &apiLabel)
	assert.Equal(t, strings.TrimLeft(dbLabel.Color, "#"), apiLabel.Color)

	// EditLabel
	newName := "LabelNewName"
	newColor := "09876a"
	newColorWrong := "09g76a"
	req = NewRequestWithJSON(t, "PATCH", singleURLStr, &api.EditLabelOption{
		Name:  &newName,
		Color: &newColor,
	}).AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)
	DecodeJSON(t, resp, &apiLabel)
	assert.Equal(t, newColor, apiLabel.Color)
	req = NewRequestWithJSON(t, "PATCH", singleURLStr, &api.EditLabelOption{
		Color: &newColorWrong,
	}).AddTokenAuth(token)
	MakeRequest(t, req, http.StatusUnprocessableEntity)

	// DeleteLabel
	req = NewRequest(t, "DELETE", singleURLStr).
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusNoContent)
}

func TestAPIAddIssueLabels(t *testing.T) {
	require.NoError(t, unittest.LoadFixtures())

	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{RepoID: repo.ID})
	_ = unittest.AssertExistsAndLoadBean(t, &issues_model.Label{RepoID: repo.ID, ID: 2})
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})

	session := loginUser(t, owner.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteIssue)
	urlStr := fmt.Sprintf("/api/v1/repos/%s/%s/issues/%d/labels",
		repo.OwnerName, repo.Name, issue.Index)
	req := NewRequestWithJSON(t, "POST", urlStr, &api.IssueLabelsOption{
		Labels: []any{1, 2},
	}).AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusOK)
	var apiLabels []*api.Label
	DecodeJSON(t, resp, &apiLabels)
	assert.Len(t, apiLabels, unittest.GetCount(t, &issues_model.IssueLabel{IssueID: issue.ID}))

	unittest.AssertExistsAndLoadBean(t, &issues_model.IssueLabel{IssueID: issue.ID, LabelID: 2})
}

func TestAPIAddIssueLabelsWithLabelNames(t *testing.T) {
	require.NoError(t, unittest.LoadFixtures())

	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 3})
	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 6, RepoID: repo.ID})
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})
	repoLabel := unittest.AssertExistsAndLoadBean(t, &issues_model.Label{ID: 10, RepoID: repo.ID})
	orgLabel := unittest.AssertExistsAndLoadBean(t, &issues_model.Label{ID: 4, OrgID: owner.ID})

	user1Session := loginUser(t, "user1")
	token := getTokenForLoggedInUser(t, user1Session, auth_model.AccessTokenScopeWriteIssue)

	// add the org label and the repo label to the issue
	urlStr := fmt.Sprintf("/api/v1/repos/%s/%s/issues/%d/labels", owner.Name, repo.Name, issue.Index)
	req := NewRequestWithJSON(t, "POST", urlStr, &api.IssueLabelsOption{
		Labels: []any{repoLabel.Name, orgLabel.Name},
	}).AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusOK)
	var apiLabels []*api.Label
	DecodeJSON(t, resp, &apiLabels)
	assert.Len(t, apiLabels, unittest.GetCount(t, &issues_model.IssueLabel{IssueID: issue.ID}))
	var apiLabelNames []string
	for _, label := range apiLabels {
		apiLabelNames = append(apiLabelNames, label.Name)
	}
	assert.ElementsMatch(t, apiLabelNames, []string{repoLabel.Name, orgLabel.Name})

	// delete labels
	req = NewRequest(t, "DELETE", urlStr).AddTokenAuth(token)
	MakeRequest(t, req, http.StatusNoContent)
}

func TestAPIRemoveIssueLabel(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{RepoID: repo.ID})
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})
	issueLabel := unittest.AssertExistsAndLoadBean(t, &issues_model.IssueLabel{IssueID: issue.ID})

	task := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionTask{ID: 47})
	task.RepoID = repo.ID
	task.OwnerID = repo.OwnerID
	require.NoError(t, task.GenerateToken())
	actions_model.UpdateTask(t.Context(), task)

	deleteURL := fmt.Sprintf("/api/v1/repos/%s/%s/issues/%d/labels/%d", owner.Name, repo.Name, issue.Index, issueLabel.LabelID)
	req := NewRequest(t, "DELETE", deleteURL).
		AddTokenAuth(task.Token)
	MakeRequest(t, req, http.StatusNoContent)

	unittest.AssertCount(t, &issues_model.IssueLabel{IssueID: issue.ID, LabelID: issueLabel.LabelID}, 0)
}

func TestAPIAddIssueLabelsAutoDate(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	issueBefore := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 3})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: issueBefore.RepoID})
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})

	session := loginUser(t, owner.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteIssue)
	urlStr := fmt.Sprintf("/api/v1/repos/%s/%s/issues/%d/labels",
		owner.Name, repo.Name, issueBefore.Index)

	t.Run("WithAutoDate", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequestWithJSON(t, "POST", urlStr, &api.IssueLabelsOption{
			Labels: []any{1},
		}).AddTokenAuth(token)
		MakeRequest(t, req, http.StatusOK)

		issueAfter := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: issueBefore.ID})
		// the execution of the API call supposedly lasted less than one minute
		updatedSince := time.Since(issueAfter.UpdatedUnix.AsTime())
		assert.LessOrEqual(t, updatedSince, time.Minute)
	})

	t.Run("WithUpdatedDate", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		updatedAt := time.Now().Add(-time.Hour).Truncate(time.Second)
		req := NewRequestWithJSON(t, "POST", urlStr, &api.IssueLabelsOption{
			Labels:  []any{2},
			Updated: &updatedAt,
		}).AddTokenAuth(token)
		MakeRequest(t, req, http.StatusOK)

		// dates will be converted into the same tz, in order to compare them
		utcTZ, _ := time.LoadLocation("UTC")
		issueAfter := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: issueBefore.ID})
		assert.Equal(t, updatedAt.In(utcTZ), issueAfter.UpdatedUnix.AsTime().In(utcTZ))
	})
}

func TestAPIReplaceIssueLabels(t *testing.T) {
	require.NoError(t, unittest.LoadFixtures())

	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{RepoID: repo.ID})
	label := unittest.AssertExistsAndLoadBean(t, &issues_model.Label{RepoID: repo.ID})
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})

	session := loginUser(t, owner.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteIssue)
	urlStr := fmt.Sprintf("/api/v1/repos/%s/%s/issues/%d/labels",
		owner.Name, repo.Name, issue.Index)
	req := NewRequestWithJSON(t, "PUT", urlStr, &api.IssueLabelsOption{
		Labels: []any{label.ID},
	}).AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusOK)
	var apiLabels []*api.Label
	DecodeJSON(t, resp, &apiLabels)
	if assert.Len(t, apiLabels, 1) {
		assert.Equal(t, label.ID, apiLabels[0].ID)
	}

	unittest.AssertCount(t, &issues_model.IssueLabel{IssueID: issue.ID}, 1)
	unittest.AssertExistsAndLoadBean(t, &issues_model.IssueLabel{IssueID: issue.ID, LabelID: label.ID})
}

func TestAPIReplaceIssueLabelsWithLabelNames(t *testing.T) {
	require.NoError(t, unittest.LoadFixtures())

	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{RepoID: repo.ID})
	label := unittest.AssertExistsAndLoadBean(t, &issues_model.Label{RepoID: repo.ID})
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})

	session := loginUser(t, owner.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteIssue)
	urlStr := fmt.Sprintf("/api/v1/repos/%s/%s/issues/%d/labels",
		owner.Name, repo.Name, issue.Index)
	req := NewRequestWithJSON(t, "PUT", urlStr, &api.IssueLabelsOption{
		Labels: []any{label.Name},
	}).AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusOK)
	var apiLabels []*api.Label
	DecodeJSON(t, resp, &apiLabels)
	if assert.Len(t, apiLabels, 1) {
		assert.Equal(t, label.Name, apiLabels[0].Name)
	}
}

func TestAPIModifyOrgLabels(t *testing.T) {
	require.NoError(t, unittest.LoadFixtures())

	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 3})
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})
	user := "user1"
	session := loginUser(t, user)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteOrganization)
	urlStr := fmt.Sprintf("/api/v1/orgs/%s/labels", owner.Name)

	// CreateLabel
	req := NewRequestWithJSON(t, "POST", urlStr, &api.CreateLabelOption{
		Name:        "TestL 1",
		Color:       "abcdef",
		Description: "test label",
	}).AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusCreated)
	apiLabel := new(api.Label)
	DecodeJSON(t, resp, &apiLabel)
	dbLabel := unittest.AssertExistsAndLoadBean(t, &issues_model.Label{ID: apiLabel.ID, OrgID: owner.ID})
	assert.Equal(t, dbLabel.Name, apiLabel.Name)
	assert.Equal(t, strings.TrimLeft(dbLabel.Color, "#"), apiLabel.Color)

	req = NewRequestWithJSON(t, "POST", urlStr, &api.CreateLabelOption{
		Name:        "TestL 2",
		Color:       "#123456",
		Description: "jet another test label",
	}).AddTokenAuth(token)
	MakeRequest(t, req, http.StatusCreated)
	req = NewRequestWithJSON(t, "POST", urlStr, &api.CreateLabelOption{
		Name:  "WrongTestL",
		Color: "#12345g",
	}).AddTokenAuth(token)
	MakeRequest(t, req, http.StatusUnprocessableEntity)

	// ListLabels
	req = NewRequest(t, "GET", urlStr).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)
	var apiLabels []*api.Label
	DecodeJSON(t, resp, &apiLabels)
	assert.Len(t, apiLabels, 4)

	// GetLabel
	singleURLStr := fmt.Sprintf("/api/v1/orgs/%s/labels/%d", owner.Name, dbLabel.ID)
	req = NewRequest(t, "GET", singleURLStr).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)
	DecodeJSON(t, resp, &apiLabel)
	assert.Equal(t, strings.TrimLeft(dbLabel.Color, "#"), apiLabel.Color)

	// EditLabel
	newName := "LabelNewName"
	newColor := "09876a"
	newColorWrong := "09g76a"
	req = NewRequestWithJSON(t, "PATCH", singleURLStr, &api.EditLabelOption{
		Name:  &newName,
		Color: &newColor,
	}).AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)
	DecodeJSON(t, resp, &apiLabel)
	assert.Equal(t, newColor, apiLabel.Color)
	req = NewRequestWithJSON(t, "PATCH", singleURLStr, &api.EditLabelOption{
		Color: &newColorWrong,
	}).AddTokenAuth(token)
	MakeRequest(t, req, http.StatusUnprocessableEntity)

	// DeleteLabel
	req = NewRequest(t, "DELETE", singleURLStr).
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusNoContent)
}
