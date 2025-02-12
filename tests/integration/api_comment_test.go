// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	auth_model "code.gitea.io/gitea/models/auth"
	"code.gitea.io/gitea/models/db"
	issues_model "code.gitea.io/gitea/models/issues"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/references"
	api "code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/services/convert"
	"code.gitea.io/gitea/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const IssueIDNotExist = 10000

func TestAPIListRepoComments(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	comment := unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{},
		unittest.Cond("type = ?", issues_model.CommentTypeComment))
	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: comment.IssueID})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: issue.RepoID})
	repoOwner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})

	link, _ := url.Parse(fmt.Sprintf("/api/v1/repos/%s/%s/issues/comments", repoOwner.Name, repo.Name))
	req := NewRequest(t, "GET", link.String())
	resp := MakeRequest(t, req, http.StatusOK)

	var apiComments []*api.Comment
	DecodeJSON(t, resp, &apiComments)
	assert.Len(t, apiComments, 3)
	for _, apiComment := range apiComments {
		c := &issues_model.Comment{ID: apiComment.ID}
		unittest.AssertExistsAndLoadBean(t, c,
			unittest.Cond("type = ?", issues_model.CommentTypeComment))
		unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: c.IssueID, RepoID: repo.ID})
	}

	// test before and since filters
	query := url.Values{}
	before := "2000-01-01T00:00:11+00:00" // unix: 946684811
	since := "2000-01-01T00:00:12+00:00"  // unix: 946684812
	query.Add("before", before)
	link.RawQuery = query.Encode()
	req = NewRequest(t, "GET", link.String())
	resp = MakeRequest(t, req, http.StatusOK)
	DecodeJSON(t, resp, &apiComments)
	assert.Len(t, apiComments, 1)
	assert.EqualValues(t, 2, apiComments[0].ID)

	query.Del("before")
	query.Add("since", since)
	link.RawQuery = query.Encode()
	req = NewRequest(t, "GET", link.String())
	resp = MakeRequest(t, req, http.StatusOK)
	DecodeJSON(t, resp, &apiComments)
	assert.Len(t, apiComments, 2)
	assert.EqualValues(t, 3, apiComments[0].ID)
}

func TestAPIListIssueComments(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	comment := unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{},
		unittest.Cond("type = ?", issues_model.CommentTypeComment))
	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: comment.IssueID})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: issue.RepoID})
	repoOwner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})

	token := getUserToken(t, repoOwner.Name, auth_model.AccessTokenScopeReadIssue)
	req := NewRequestf(t, "GET", "/api/v1/repos/%s/%s/issues/%d/comments", repoOwner.Name, repo.Name, issue.Index).
		AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusOK)

	var comments []*api.Comment
	DecodeJSON(t, resp, &comments)
	expectedCount := unittest.GetCount(t, &issues_model.Comment{IssueID: issue.ID},
		unittest.Cond("type = ?", issues_model.CommentTypeComment))
	assert.Len(t, comments, expectedCount)

	req = NewRequestf(t, "GET", "/api/v1/repos/%s/%s/issues/%d/comments", repoOwner.Name, repo.Name, IssueIDNotExist).
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusNotFound)
}

func TestAPICreateComment(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	const commentBody = "Comment body"

	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: issue.RepoID})
	repoOwner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})

	token := getUserToken(t, repoOwner.Name, auth_model.AccessTokenScopeWriteIssue)
	urlStr := fmt.Sprintf("/api/v1/repos/%s/%s/issues/%d/comments",
		repoOwner.Name, repo.Name, issue.Index)
	req := NewRequestWithValues(t, "POST", urlStr, map[string]string{
		"body": commentBody,
	}).AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusCreated)

	var updatedComment api.Comment
	DecodeJSON(t, resp, &updatedComment)
	assert.EqualValues(t, commentBody, updatedComment.Body)
	unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{ID: updatedComment.ID, IssueID: issue.ID, Content: commentBody})

	urlStr = fmt.Sprintf("/api/v1/repos/%s/%s/issues/%d/comments",
		repoOwner.Name, repo.Name, IssueIDNotExist)
	req = NewRequestWithValues(t, "POST", urlStr, map[string]string{
		"body": commentBody,
	}).AddTokenAuth(token)
	MakeRequest(t, req, http.StatusNotFound)
}

func TestAPICreateCommentAutoDate(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: issue.RepoID})
	repoOwner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})
	token := getUserToken(t, repoOwner.Name, auth_model.AccessTokenScopeWriteIssue)

	urlStr := fmt.Sprintf("/api/v1/repos/%s/%s/issues/%d/comments",
		repoOwner.Name, repo.Name, issue.Index)
	const commentBody = "Comment body"

	t.Run("WithAutoDate", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequestWithValues(t, "POST", urlStr, map[string]string{
			"body": commentBody,
		}).AddTokenAuth(token)
		resp := MakeRequest(t, req, http.StatusCreated)
		var updatedComment api.Comment
		DecodeJSON(t, resp, &updatedComment)

		// the execution of the API call supposedly lasted less than one minute
		updatedSince := time.Since(updatedComment.Updated)
		assert.LessOrEqual(t, updatedSince, time.Minute)

		commentAfter := unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{ID: updatedComment.ID, IssueID: issue.ID, Content: commentBody})
		updatedSince = time.Since(commentAfter.UpdatedUnix.AsTime())
		assert.LessOrEqual(t, updatedSince, time.Minute)
	})

	t.Run("WithUpdateDate", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		updatedAt := time.Now().Add(-time.Hour).Truncate(time.Second)

		req := NewRequestWithJSON(t, "POST", urlStr, &api.CreateIssueCommentOption{
			Body:    commentBody,
			Updated: &updatedAt,
		}).AddTokenAuth(token)
		resp := MakeRequest(t, req, http.StatusCreated)
		var updatedComment api.Comment
		DecodeJSON(t, resp, &updatedComment)

		// dates will be converted into the same tz, in order to compare them
		utcTZ, _ := time.LoadLocation("UTC")
		assert.Equal(t, updatedAt.In(utcTZ), updatedComment.Updated.In(utcTZ))
		commentAfter := unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{ID: updatedComment.ID, IssueID: issue.ID, Content: commentBody})
		assert.Equal(t, updatedAt.In(utcTZ), commentAfter.UpdatedUnix.AsTime().In(utcTZ))
	})
}

func TestAPICommentXRefAutoDate(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 1})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: issue.RepoID})
	repoOwner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})
	token := getUserToken(t, repoOwner.Name, auth_model.AccessTokenScopeWriteIssue)

	t.Run("WithAutoDate", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		// Create a comment mentioning issue #2 and check that a xref comment was added
		// in issue #2
		urlStr := fmt.Sprintf("/api/v1/repos/%s/%s/issues/%d/comments",
			repoOwner.Name, repo.Name, issue.Index)

		commentBody := "mention #2"
		req := NewRequestWithJSON(t, "POST", urlStr, &api.CreateIssueCommentOption{
			Body: commentBody,
		}).AddTokenAuth(token)
		resp := MakeRequest(t, req, http.StatusCreated)
		var createdComment api.Comment
		DecodeJSON(t, resp, &createdComment)

		ref := unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{IssueID: 2, RefIssueID: 1, RefCommentID: createdComment.ID})
		assert.Equal(t, issues_model.CommentTypeCommentRef, ref.Type)
		assert.Equal(t, references.XRefActionNone, ref.RefAction)
		// the execution of the API call supposedly lasted less than one minute
		updatedSince := time.Since(ref.UpdatedUnix.AsTime())
		assert.LessOrEqual(t, updatedSince, time.Minute)

		// Remove the mention to issue #2 and check that the xref was neutered
		urlStr = fmt.Sprintf("/api/v1/repos/%s/%s/issues/comments/%d",
			repoOwner.Name, repo.Name, createdComment.ID)

		newCommentBody := "no mention"
		req = NewRequestWithJSON(t, "PATCH", urlStr, &api.EditIssueCommentOption{
			Body: newCommentBody,
		}).AddTokenAuth(token)
		resp = MakeRequest(t, req, http.StatusOK)
		var updatedComment api.Comment
		DecodeJSON(t, resp, &updatedComment)

		ref = unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{IssueID: 2, RefIssueID: 1, RefCommentID: updatedComment.ID})
		assert.Equal(t, issues_model.CommentTypeCommentRef, ref.Type)
		assert.Equal(t, references.XRefActionNeutered, ref.RefAction)
		// the execution of the API call supposedly lasted less than one minute
		updatedSince = time.Since(ref.UpdatedUnix.AsTime())
		assert.LessOrEqual(t, updatedSince, time.Minute)
	})

	t.Run("WithUpdateDate", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		// dates will be converted into the same tz, in order to compare them
		utcTZ, _ := time.LoadLocation("UTC")

		// Create a comment mentioning issue #2 and check that a xref comment was added
		// in issue #2
		urlStr := fmt.Sprintf("/api/v1/repos/%s/%s/issues/%d/comments",
			repoOwner.Name, repo.Name, issue.Index)

		commentBody := "re-mention #2"
		updatedAt := time.Now().Add(-time.Hour).Truncate(time.Second)
		req := NewRequestWithJSON(t, "POST", urlStr, &api.CreateIssueCommentOption{
			Body:    commentBody,
			Updated: &updatedAt,
		}).AddTokenAuth(token)
		resp := MakeRequest(t, req, http.StatusCreated)
		var createdComment api.Comment
		DecodeJSON(t, resp, &createdComment)

		ref := unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{IssueID: 2, RefIssueID: 1, RefCommentID: createdComment.ID})
		assert.Equal(t, issues_model.CommentTypeCommentRef, ref.Type)
		assert.Equal(t, references.XRefActionNone, ref.RefAction)
		assert.Equal(t, updatedAt.In(utcTZ), ref.UpdatedUnix.AsTimeInLocation(utcTZ))

		// Remove the mention to issue #2 and check that the xref was neutered
		urlStr = fmt.Sprintf("/api/v1/repos/%s/%s/issues/comments/%d",
			repoOwner.Name, repo.Name, createdComment.ID)

		newCommentBody := "no mention"
		updatedAt = time.Now().Add(-time.Hour).Truncate(time.Second)
		req = NewRequestWithJSON(t, "PATCH", urlStr, &api.EditIssueCommentOption{
			Body:    newCommentBody,
			Updated: &updatedAt,
		}).AddTokenAuth(token)
		resp = MakeRequest(t, req, http.StatusOK)
		var updatedComment api.Comment
		DecodeJSON(t, resp, &updatedComment)

		ref = unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{IssueID: 2, RefIssueID: 1, RefCommentID: updatedComment.ID})
		assert.Equal(t, issues_model.CommentTypeCommentRef, ref.Type)
		assert.Equal(t, references.XRefActionNeutered, ref.RefAction)
		assert.Equal(t, updatedAt.In(utcTZ), ref.UpdatedUnix.AsTimeInLocation(utcTZ))
	})
}

func TestAPIGetComment(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	comment := unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{ID: 2})
	require.NoError(t, comment.LoadIssue(db.DefaultContext))
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: comment.Issue.RepoID})
	repoOwner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})

	token := getUserToken(t, repoOwner.Name, auth_model.AccessTokenScopeReadIssue)
	req := NewRequestf(t, "GET", "/api/v1/repos/%s/%s/issues/comments/%d", repoOwner.Name, repo.Name, comment.ID)
	MakeRequest(t, req, http.StatusOK)
	req = NewRequestf(t, "GET", "/api/v1/repos/%s/%s/issues/comments/%d", repoOwner.Name, repo.Name, comment.ID).
		AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusOK)

	var apiComment api.Comment
	DecodeJSON(t, resp, &apiComment)

	require.NoError(t, comment.LoadPoster(db.DefaultContext))
	expect := convert.ToAPIComment(db.DefaultContext, repo, comment)

	assert.Equal(t, expect.ID, apiComment.ID)
	assert.Equal(t, expect.Poster.FullName, apiComment.Poster.FullName)
	assert.Equal(t, expect.Body, apiComment.Body)
	assert.Equal(t, expect.Created.Unix(), apiComment.Created.Unix())
}

func TestAPIGetSystemUserComment(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: issue.RepoID})
	repoOwner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})

	for _, systemUser := range []*user_model.User{
		user_model.NewGhostUser(),
		user_model.NewActionsUser(),
	} {
		body := fmt.Sprintf("Hello %s", systemUser.Name)
		comment, err := issues_model.CreateComment(db.DefaultContext, &issues_model.CreateCommentOptions{
			Type:    issues_model.CommentTypeComment,
			Doer:    systemUser,
			Repo:    repo,
			Issue:   issue,
			Content: body,
		})
		require.NoError(t, err)

		req := NewRequestf(t, "GET", "/api/v1/repos/%s/%s/issues/comments/%d", repoOwner.Name, repo.Name, comment.ID)
		resp := MakeRequest(t, req, http.StatusOK)

		var apiComment api.Comment
		DecodeJSON(t, resp, &apiComment)

		if assert.NotNil(t, apiComment.Poster) {
			if assert.Equal(t, systemUser.ID, apiComment.Poster.ID) {
				require.NoError(t, comment.LoadPoster(db.DefaultContext))
				assert.Equal(t, systemUser.Name, apiComment.Poster.UserName)
			}
		}
		assert.Equal(t, body, apiComment.Body)
	}
}

func TestAPIEditComment(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	const newCommentBody = "This is the new comment body"

	comment := unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{ID: 8},
		unittest.Cond("type = ?", issues_model.CommentTypeComment))
	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: comment.IssueID})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: issue.RepoID})
	repoOwner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})

	t.Run("UnrelatedCommentID", func(t *testing.T) {
		// Using the ID of a comment that does not belong to the repository must fail
		repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 4})
		repoOwner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})
		token := getUserToken(t, repoOwner.Name, auth_model.AccessTokenScopeWriteIssue)
		urlStr := fmt.Sprintf("/api/v1/repos/%s/%s/issues/comments/%d",
			repoOwner.Name, repo.Name, comment.ID)
		req := NewRequestWithValues(t, "PATCH", urlStr, map[string]string{
			"body": newCommentBody,
		}).AddTokenAuth(token)
		MakeRequest(t, req, http.StatusNotFound)
	})

	token := getUserToken(t, repoOwner.Name, auth_model.AccessTokenScopeWriteIssue)
	urlStr := fmt.Sprintf("/api/v1/repos/%s/%s/issues/comments/%d",
		repoOwner.Name, repo.Name, comment.ID)
	req := NewRequestWithValues(t, "PATCH", urlStr, map[string]string{
		"body": newCommentBody,
	}).AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusOK)

	var updatedComment api.Comment
	DecodeJSON(t, resp, &updatedComment)
	assert.EqualValues(t, comment.ID, updatedComment.ID)
	assert.EqualValues(t, newCommentBody, updatedComment.Body)
	unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{ID: comment.ID, IssueID: issue.ID, Content: newCommentBody})
}

func TestAPIEditCommentWithDate(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	comment := unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{},
		unittest.Cond("type = ?", issues_model.CommentTypeComment))
	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: comment.IssueID})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: issue.RepoID})
	repoOwner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})
	token := getUserToken(t, repoOwner.Name, auth_model.AccessTokenScopeWriteIssue)

	urlStr := fmt.Sprintf("/api/v1/repos/%s/%s/issues/comments/%d",
		repoOwner.Name, repo.Name, comment.ID)
	const newCommentBody = "This is the new comment body"

	t.Run("WithAutoDate", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequestWithValues(t, "PATCH", urlStr, map[string]string{
			"body": newCommentBody,
		}).AddTokenAuth(token)
		resp := MakeRequest(t, req, http.StatusOK)
		var updatedComment api.Comment
		DecodeJSON(t, resp, &updatedComment)

		// the execution of the API call supposedly lasted less than one minute
		updatedSince := time.Since(updatedComment.Updated)
		assert.LessOrEqual(t, updatedSince, time.Minute)

		commentAfter := unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{ID: comment.ID, IssueID: issue.ID, Content: newCommentBody})
		updatedSince = time.Since(commentAfter.UpdatedUnix.AsTime())
		assert.LessOrEqual(t, updatedSince, time.Minute)
	})

	t.Run("WithUpdateDate", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		updatedAt := time.Now().Add(-time.Hour).Truncate(time.Second)

		req := NewRequestWithJSON(t, "PATCH", urlStr, &api.EditIssueCommentOption{
			Body:    newCommentBody,
			Updated: &updatedAt,
		}).AddTokenAuth(token)
		resp := MakeRequest(t, req, http.StatusOK)
		var updatedComment api.Comment
		DecodeJSON(t, resp, &updatedComment)

		// dates will be converted into the same tz, in order to compare them
		utcTZ, _ := time.LoadLocation("UTC")
		assert.Equal(t, updatedAt.In(utcTZ), updatedComment.Updated.In(utcTZ))
		commentAfter := unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{ID: comment.ID, IssueID: issue.ID, Content: newCommentBody})
		assert.Equal(t, updatedAt.In(utcTZ), commentAfter.UpdatedUnix.AsTime().In(utcTZ))
	})
}

func TestAPIDeleteComment(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	comment := unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{ID: 8},
		unittest.Cond("type = ?", issues_model.CommentTypeComment))
	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: comment.IssueID})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: issue.RepoID})
	repoOwner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})

	t.Run("UnrelatedCommentID", func(t *testing.T) {
		// Using the ID of a comment that does not belong to the repository must fail
		repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 4})
		repoOwner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})
		token := getUserToken(t, repoOwner.Name, auth_model.AccessTokenScopeWriteIssue)
		req := NewRequestf(t, "DELETE", "/api/v1/repos/%s/%s/issues/comments/%d", repoOwner.Name, repo.Name, comment.ID).
			AddTokenAuth(token)
		MakeRequest(t, req, http.StatusNotFound)
	})

	token := getUserToken(t, repoOwner.Name, auth_model.AccessTokenScopeWriteIssue)
	req := NewRequestf(t, "DELETE", "/api/v1/repos/%s/%s/issues/comments/%d", repoOwner.Name, repo.Name, comment.ID).
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusNoContent)

	unittest.AssertNotExistsBean(t, &issues_model.Comment{ID: comment.ID})
}

func TestAPIListIssueTimeline(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// load comment
	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 1})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: issue.RepoID})
	repoOwner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})

	// make request
	req := NewRequestf(t, "GET", "/api/v1/repos/%s/%s/issues/%d/timeline", repoOwner.Name, repo.Name, issue.Index)
	resp := MakeRequest(t, req, http.StatusOK)

	// check if lens of list returned by API and
	// lists extracted directly from DB are the same
	var comments []*api.TimelineComment
	DecodeJSON(t, resp, &comments)
	expectedCount := unittest.GetCount(t, &issues_model.Comment{IssueID: issue.ID})
	assert.Len(t, comments, expectedCount)

	req = NewRequestf(t, "GET", "/api/v1/repos/%s/%s/issues/%d/timeline", repoOwner.Name, repo.Name, IssueIDNotExist)
	MakeRequest(t, req, http.StatusNotFound)
}
