// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"testing"
	"time"

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

func TestAPIGetIssueAttachment(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	attachment := unittest.AssertExistsAndLoadBean(t, &repo_model.Attachment{ID: 1})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: attachment.RepoID})
	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{RepoID: attachment.IssueID})
	repoOwner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})

	session := loginUser(t, repoOwner.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeReadIssue)

	req := NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/issues/%d/assets/%d", repoOwner.Name, repo.Name, issue.Index, attachment.ID)).
		AddTokenAuth(token)
	resp := session.MakeRequest(t, req, http.StatusOK)
	apiAttachment := new(api.Attachment)
	DecodeJSON(t, resp, &apiAttachment)

	unittest.AssertExistsAndLoadBean(t, &repo_model.Attachment{ID: apiAttachment.ID, IssueID: issue.ID})
}

func TestAPIListIssueAttachments(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	attachment := unittest.AssertExistsAndLoadBean(t, &repo_model.Attachment{ID: 1})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: attachment.RepoID})
	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{RepoID: attachment.IssueID})
	repoOwner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})

	session := loginUser(t, repoOwner.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeReadIssue)

	req := NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/issues/%d/assets", repoOwner.Name, repo.Name, issue.Index)).
		AddTokenAuth(token)
	resp := session.MakeRequest(t, req, http.StatusOK)
	apiAttachment := new([]api.Attachment)
	DecodeJSON(t, resp, &apiAttachment)

	unittest.AssertExistsAndLoadBean(t, &repo_model.Attachment{ID: (*apiAttachment)[0].ID, IssueID: issue.ID})
}

func TestAPICreateIssueAttachment(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{RepoID: repo.ID})
	repoOwner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})

	session := loginUser(t, repoOwner.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteIssue)

	filename := "image.png"
	buff := generateImg()
	body := &bytes.Buffer{}

	// Setup multi-part
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("attachment", filename)
	require.NoError(t, err)
	_, err = io.Copy(part, &buff)
	require.NoError(t, err)
	err = writer.Close()
	require.NoError(t, err)

	req := NewRequestWithBody(t, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/issues/%d/assets", repoOwner.Name, repo.Name, issue.Index), body).
		AddTokenAuth(token)
	req.Header.Add("Content-Type", writer.FormDataContentType())
	resp := session.MakeRequest(t, req, http.StatusCreated)

	apiAttachment := new(api.Attachment)
	DecodeJSON(t, resp, &apiAttachment)

	unittest.AssertExistsAndLoadBean(t, &repo_model.Attachment{ID: apiAttachment.ID, IssueID: issue.ID})
}

func TestAPICreateIssueAttachmentAutoDate(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{RepoID: repo.ID})
	repoOwner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})

	session := loginUser(t, repoOwner.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteIssue)
	urlStr := fmt.Sprintf("/api/v1/repos/%s/%s/issues/%d/assets",
		repoOwner.Name, repo.Name, issue.Index)

	filename := "image.png"
	buff := generateImg()
	body := &bytes.Buffer{}

	t.Run("WithAutoDate", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		// Setup multi-part
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("attachment", filename)
		require.NoError(t, err)
		_, err = io.Copy(part, &buff)
		require.NoError(t, err)
		err = writer.Close()
		require.NoError(t, err)

		req := NewRequestWithBody(t, "POST", urlStr, body).AddTokenAuth(token)
		req.Header.Add("Content-Type", writer.FormDataContentType())
		resp := session.MakeRequest(t, req, http.StatusCreated)

		apiAttachment := new(api.Attachment)
		DecodeJSON(t, resp, &apiAttachment)

		unittest.AssertExistsAndLoadBean(t, &repo_model.Attachment{ID: apiAttachment.ID, IssueID: issue.ID})
		// the execution of the API call supposedly lasted less than one minute
		updatedSince := time.Since(apiAttachment.Created)
		assert.LessOrEqual(t, updatedSince, time.Minute)

		issueAfter := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: issue.Index})
		updatedSince = time.Since(issueAfter.UpdatedUnix.AsTime())
		assert.LessOrEqual(t, updatedSince, time.Minute)
	})

	t.Run("WithUpdateDate", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		updatedAt := time.Now().Add(-time.Hour).Truncate(time.Second)
		urlStr += fmt.Sprintf("?updated_at=%s", updatedAt.UTC().Format(time.RFC3339))

		// Setup multi-part
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("attachment", filename)
		require.NoError(t, err)
		_, err = io.Copy(part, &buff)
		require.NoError(t, err)
		err = writer.Close()
		require.NoError(t, err)

		req := NewRequestWithBody(t, "POST", urlStr, body).AddTokenAuth(token)
		req.Header.Add("Content-Type", writer.FormDataContentType())
		resp := session.MakeRequest(t, req, http.StatusCreated)

		apiAttachment := new(api.Attachment)
		DecodeJSON(t, resp, &apiAttachment)

		// dates will be converted into the same tz, in order to compare them
		utcTZ, _ := time.LoadLocation("UTC")
		unittest.AssertExistsAndLoadBean(t, &repo_model.Attachment{ID: apiAttachment.ID, IssueID: issue.ID})
		assert.Equal(t, updatedAt.In(utcTZ), apiAttachment.Created.In(utcTZ))
		issueAfter := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: issue.ID})
		assert.Equal(t, updatedAt.In(utcTZ), issueAfter.UpdatedUnix.AsTime().In(utcTZ))
	})
}

func TestAPICreateIssueAttachmentWithUnallowedFile(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{RepoID: repo.ID})
	repoOwner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})

	session := loginUser(t, repoOwner.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteIssue)

	filename := "file.bad"
	body := &bytes.Buffer{}

	// Setup multi-part.
	writer := multipart.NewWriter(body)
	_, err := writer.CreateFormFile("attachment", filename)
	require.NoError(t, err)
	err = writer.Close()
	require.NoError(t, err)

	req := NewRequestWithBody(t, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/issues/%d/assets", repoOwner.Name, repo.Name, issue.Index), body).
		AddTokenAuth(token)
	req.Header.Add("Content-Type", writer.FormDataContentType())

	session.MakeRequest(t, req, http.StatusUnprocessableEntity)
}

func TestAPIEditIssueAttachment(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	const newAttachmentName = "newAttachmentName"

	attachment := unittest.AssertExistsAndLoadBean(t, &repo_model.Attachment{ID: 1})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: attachment.RepoID})
	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{RepoID: attachment.IssueID})
	repoOwner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})

	session := loginUser(t, repoOwner.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteIssue)
	urlStr := fmt.Sprintf("/api/v1/repos/%s/%s/issues/%d/assets/%d",
		repoOwner.Name, repo.Name, issue.Index, attachment.ID)
	req := NewRequestWithValues(t, "PATCH", urlStr, map[string]string{
		"name": newAttachmentName,
	}).AddTokenAuth(token)
	resp := session.MakeRequest(t, req, http.StatusCreated)
	apiAttachment := new(api.Attachment)
	DecodeJSON(t, resp, &apiAttachment)

	unittest.AssertExistsAndLoadBean(t, &repo_model.Attachment{ID: apiAttachment.ID, IssueID: issue.ID, Name: apiAttachment.Name})
}

func TestAPIDeleteIssueAttachment(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	attachment := unittest.AssertExistsAndLoadBean(t, &repo_model.Attachment{ID: 1})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: attachment.RepoID})
	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{RepoID: attachment.IssueID})
	repoOwner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})

	session := loginUser(t, repoOwner.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteIssue)

	req := NewRequest(t, "DELETE", fmt.Sprintf("/api/v1/repos/%s/%s/issues/%d/assets/%d", repoOwner.Name, repo.Name, issue.Index, attachment.ID)).
		AddTokenAuth(token)
	session.MakeRequest(t, req, http.StatusNoContent)

	unittest.AssertNotExistsBean(t, &repo_model.Attachment{ID: attachment.ID, IssueID: issue.ID})
}
