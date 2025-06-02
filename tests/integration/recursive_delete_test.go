// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT
package integration

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	auth_model "forgejo.org/models/auth"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	api "forgejo.org/modules/structs"

	"github.com/stretchr/testify/assert"
)

func getCreateOptionsFile1() api.CreateFileOptions {
	content := "This is new text file1.txt"
	contentEncoded := base64.StdEncoding.EncodeToString([]byte(content))
	return api.CreateFileOptions{
		FileOptions: api.FileOptions{
			BranchName:    "master",
			NewBranchName: "master",
			Message:       "file1.txt",
			Author: api.Identity{
				Name:  "Anne Doe",
				Email: "annedoe@example.com",
			},
			Committer: api.Identity{
				Name:  "John Doe",
				Email: "johndoe@example.com",
			},
			Dates: api.CommitDateOptions{
				Author:    time.Unix(946684810, 0),
				Committer: time.Unix(978307190, 0),
			},
		},
		ContentBase64: contentEncoded,
	}
}

func getCreateOptionsFile2() api.CreateFileOptions {
	content := "This is new text dir2/file2.txt"
	contentEncoded := base64.StdEncoding.EncodeToString([]byte(content))
	return api.CreateFileOptions{
		FileOptions: api.FileOptions{
			BranchName:    "master",
			NewBranchName: "master",
			Message:       "dir2/file2.txt",
			Author: api.Identity{
				Name:  "Anne Doe",
				Email: "annedoe@example.com",
			},
			Committer: api.Identity{
				Name:  "John Doe",
				Email: "johndoe@example.com",
			},
			Dates: api.CommitDateOptions{
				Author:    time.Unix(946684810, 0),
				Committer: time.Unix(978307190, 0),
			},
		},
		ContentBase64: contentEncoded,
	}
}

func getCreateOptionsFile3() api.CreateFileOptions {
	content := "This is new text dir2/dir3/file3.txt"
	contentEncoded := base64.StdEncoding.EncodeToString([]byte(content))
	return api.CreateFileOptions{
		FileOptions: api.FileOptions{
			BranchName:    "master",
			NewBranchName: "master",
			Message:       "dir2/dir3/file3.txt",
			Author: api.Identity{
				Name:  "Anne Doe",
				Email: "annedoe@example.com",
			},
			Committer: api.Identity{
				Name:  "John Doe",
				Email: "johndoe@example.com",
			},
			Dates: api.CommitDateOptions{
				Author:    time.Unix(946684810, 0),
				Committer: time.Unix(978307190, 0),
			},
		},
		ContentBase64: contentEncoded,
	}
}

func getCreateOptionsFile4() api.CreateFileOptions {
	content := "This is new text dir2/dir4/file4.txt"
	contentEncoded := base64.StdEncoding.EncodeToString([]byte(content))
	return api.CreateFileOptions{
		FileOptions: api.FileOptions{
			BranchName:    "master",
			NewBranchName: "master",
			Message:       "dir2/dir3/file4.txt",
			Author: api.Identity{
				Name:  "Anne Doe",
				Email: "annedoe@example.com",
			},
			Committer: api.Identity{
				Name:  "John Doe",
				Email: "johndoe@example.com",
			},
			Dates: api.CommitDateOptions{
				Author:    time.Unix(946684810, 0),
				Committer: time.Unix(978307190, 0),
			},
		},
		ContentBase64: contentEncoded,
	}
}

// Structure:
// file1.txt
// dir2/file2.txt
// dir2/dir3/file3.txt
// dir2/dir4/file4.txt
// delete: dir2/dir3
func TestRecursiveDeleteSubSub(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})       // owner of the repo1
		repo1 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1}) // public repo

		// Get user2's token
		session := loginUser(t, user2.Name)
		token2 := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)

		createFileOptions1 := getCreateOptionsFile1()
		treePath1 := "file1.txt"
		req1 := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user2.Name, repo1.Name, treePath1), &createFileOptions1).AddTokenAuth(token2)
		resp1 := MakeRequest(t, req1, http.StatusCreated)
		var fileResponse1 api.FileResponse
		DecodeJSON(t, resp1, &fileResponse1)
		assert.Equal(t, fileResponse1.Content.Path, treePath1)
		assert.EqualValues(t, 26, fileResponse1.Content.Size)

		createFileOptions2 := getCreateOptionsFile2()
		treePath2 := "dir2/file2.txt"
		req2 := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user2.Name, repo1.Name, treePath2), &createFileOptions2).AddTokenAuth(token2)
		resp2 := MakeRequest(t, req2, http.StatusCreated)
		var fileResponse2 api.FileResponse
		DecodeJSON(t, resp2, &fileResponse2)
		assert.Equal(t, fileResponse2.Content.Path, treePath2)
		assert.EqualValues(t, 31, fileResponse2.Content.Size)

		createFileOptions3 := getCreateOptionsFile3()
		treePath3 := "dir2/dir3/file3.txt"
		req3 := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user2.Name, repo1.Name, treePath3), &createFileOptions3).AddTokenAuth(token2)
		resp3 := MakeRequest(t, req3, http.StatusCreated)
		var fileResponse3 api.FileResponse
		DecodeJSON(t, resp3, &fileResponse3)
		assert.Equal(t, fileResponse3.Content.Path, treePath3)
		assert.EqualValues(t, 36, fileResponse3.Content.Size)

		createFileOptions4 := getCreateOptionsFile4()
		treePath4 := "dir2/dir4/file4.txt"
		req4 := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user2.Name, repo1.Name, treePath4), &createFileOptions4).AddTokenAuth(token2)
		resp4 := MakeRequest(t, req4, http.StatusCreated)
		var fileResponse4 api.FileResponse
		DecodeJSON(t, resp4, &fileResponse4)
		assert.Equal(t, fileResponse4.Content.Path, treePath4)
		assert.EqualValues(t, 36, fileResponse4.Content.Size)

		// Verify file1 exists
		getReq1 := NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user2.Name, repo1.Name, treePath1))
		MakeRequest(t, getReq1, http.StatusOK)

		// Verify file2 exists
		getReq2 := NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user2.Name, repo1.Name, treePath2))
		MakeRequest(t, getReq2, http.StatusOK)

		// Verify file3 exists
		getReq3 := NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user2.Name, repo1.Name, treePath3))
		MakeRequest(t, getReq3, http.StatusOK)

		// Verify file4 exists
		getReq4 := NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user2.Name, repo1.Name, treePath4))
		MakeRequest(t, getReq4, http.StatusOK)

		treePathDirDel := "dir2/dir3"
		req := NewRequest(t, "GET", fmt.Sprintf("/%s/%s/_delete_path/master/%s", user2.Name, repo1.Name, treePathDirDel))
		session.MakeRequest(t, req, http.StatusOK)

		csrf := GetCSRF(t, session, fmt.Sprintf("/%s/%s/_delete_path/master/%s", user2.Name, repo1.Name, treePathDirDel))

		commitForm := map[string]string{
			"_csrf":          csrf,
			"commit_summary": "Delete",
			"commit_message": "",
			"commit_choice":  "direct",
			"commit_mail_id": "-1",
		}

		// POST to the commit endpoint
		postReq := NewRequestWithValues(t, "POST", fmt.Sprintf("/%s/%s/_delete_path/master/%s", user2.Name, repo1.Name, treePathDirDel), commitForm)
		postResp := session.MakeRequest(t, postReq, http.StatusSeeOther)

		redirectLocation := postResp.Header().Get("Location")
		if redirectLocation != "" {
			verifyReq := NewRequest(t, "GET", redirectLocation)
			session.MakeRequest(t, verifyReq, http.StatusOK)
		}

		// Verify file1 exists
		getReq1 = NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user2.Name, repo1.Name, treePath1))
		MakeRequest(t, getReq1, http.StatusOK)

		// Verify file2 exists
		getReq2 = NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user2.Name, repo1.Name, treePath2))
		MakeRequest(t, getReq2, http.StatusOK)

		// Verify file3 exists
		getReq3 = NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user2.Name, repo1.Name, treePath3))
		MakeRequest(t, getReq3, http.StatusNotFound)

		// Verify file4 exists
		getReq4 = NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user2.Name, repo1.Name, treePath4))
		MakeRequest(t, getReq4, http.StatusOK)
	})
}

// Structure:
// file1.txt
// dir2/file2.txt
// dir2/dir3/file3.txt
// dir2/dir4/file4.txt
// delete: dir2
func TestRecursiveDeleteSub(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})       // owner of the repo1
		repo1 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1}) // public repo

		// Get user2's token
		session := loginUser(t, user2.Name)
		token2 := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)

		createFileOptions1 := getCreateOptionsFile1()
		treePath1 := "file1.txt"
		req1 := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user2.Name, repo1.Name, treePath1), &createFileOptions1).AddTokenAuth(token2)
		resp1 := MakeRequest(t, req1, http.StatusCreated)
		var fileResponse1 api.FileResponse
		DecodeJSON(t, resp1, &fileResponse1)
		assert.Equal(t, fileResponse1.Content.Path, treePath1)
		assert.EqualValues(t, 26, fileResponse1.Content.Size)

		createFileOptions2 := getCreateOptionsFile2()
		treePath2 := "dir2/file2.txt"
		req2 := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user2.Name, repo1.Name, treePath2), &createFileOptions2).AddTokenAuth(token2)
		resp2 := MakeRequest(t, req2, http.StatusCreated)
		var fileResponse2 api.FileResponse
		DecodeJSON(t, resp2, &fileResponse2)
		assert.Equal(t, fileResponse2.Content.Path, treePath2)
		assert.EqualValues(t, 31, fileResponse2.Content.Size)

		createFileOptions3 := getCreateOptionsFile3()
		treePath3 := "dir2/dir3/file3.txt"
		req3 := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user2.Name, repo1.Name, treePath3), &createFileOptions3).AddTokenAuth(token2)
		resp3 := MakeRequest(t, req3, http.StatusCreated)
		var fileResponse3 api.FileResponse
		DecodeJSON(t, resp3, &fileResponse3)
		assert.Equal(t, fileResponse3.Content.Path, treePath3)
		assert.EqualValues(t, 36, fileResponse3.Content.Size)

		createFileOptions4 := getCreateOptionsFile4()
		treePath4 := "dir2/dir4/file4.txt"
		req4 := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user2.Name, repo1.Name, treePath4), &createFileOptions4).AddTokenAuth(token2)
		resp4 := MakeRequest(t, req4, http.StatusCreated)
		var fileResponse4 api.FileResponse
		DecodeJSON(t, resp4, &fileResponse4)
		assert.Equal(t, fileResponse4.Content.Path, treePath4)
		assert.EqualValues(t, 36, fileResponse4.Content.Size)

		// Verify file1 exists
		getReq1 := NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user2.Name, repo1.Name, treePath1))
		MakeRequest(t, getReq1, http.StatusOK)

		// Verify file2 exists
		getReq2 := NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user2.Name, repo1.Name, treePath2))
		MakeRequest(t, getReq2, http.StatusOK)

		// Verify file3 exists
		getReq3 := NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user2.Name, repo1.Name, treePath3))
		MakeRequest(t, getReq3, http.StatusOK)

		// Verify file4 exists
		getReq4 := NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user2.Name, repo1.Name, treePath4))
		MakeRequest(t, getReq4, http.StatusOK)

		treePathDirDel := "dir2"
		req := NewRequest(t, "GET", fmt.Sprintf("/%s/%s/_delete_path/master/%s", user2.Name, repo1.Name, treePathDirDel))
		session.MakeRequest(t, req, http.StatusOK)

		csrf := GetCSRF(t, session, fmt.Sprintf("/%s/%s/_delete_path/master/%s", user2.Name, repo1.Name, treePathDirDel))

		commitForm := map[string]string{
			"_csrf":          csrf,
			"commit_summary": "Delete",
			"commit_message": "",
			"commit_choice":  "direct",
			"commit_mail_id": "-1",
		}

		// POST to the commit endpoint
		postReq := NewRequestWithValues(t, "POST", fmt.Sprintf("/%s/%s/_delete_path/master/%s", user2.Name, repo1.Name, treePathDirDel), commitForm)
		postResp := session.MakeRequest(t, postReq, http.StatusSeeOther)

		redirectLocation := postResp.Header().Get("Location")
		if redirectLocation != "" {
			verifyReq := NewRequest(t, "GET", redirectLocation)
			session.MakeRequest(t, verifyReq, http.StatusOK)
		}

		// Verify file1 exists
		getReq1 = NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user2.Name, repo1.Name, treePath1))
		MakeRequest(t, getReq1, http.StatusOK)

		// Verify file2 exists
		getReq2 = NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user2.Name, repo1.Name, treePath2))
		MakeRequest(t, getReq2, http.StatusNotFound)

		// Verify file3 exists
		getReq3 = NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user2.Name, repo1.Name, treePath3))
		MakeRequest(t, getReq3, http.StatusNotFound)

		// Verify file4 exists
		getReq4 = NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user2.Name, repo1.Name, treePath4))
		MakeRequest(t, getReq4, http.StatusNotFound)
	})
}

// Structure:
// file1.txt
// dir2/file2.txt
// dir2/dir3/file3.txt
// dir2/dir4/file4.txt
// delete: ROOT
func TestRecursiveDeleteRoot(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})       // owner of the repo1
		repo1 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1}) // public repo

		// Get user2's token
		session := loginUser(t, user2.Name)
		token2 := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)

		createFileOptions1 := getCreateOptionsFile1()
		treePath1 := "file1.txt"
		req1 := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user2.Name, repo1.Name, treePath1), &createFileOptions1).AddTokenAuth(token2)
		resp1 := MakeRequest(t, req1, http.StatusCreated)
		var fileResponse1 api.FileResponse
		DecodeJSON(t, resp1, &fileResponse1)
		assert.Equal(t, fileResponse1.Content.Path, treePath1)
		assert.EqualValues(t, 26, fileResponse1.Content.Size)

		createFileOptions2 := getCreateOptionsFile2()
		treePath2 := "dir2/file2.txt"
		req2 := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user2.Name, repo1.Name, treePath2), &createFileOptions2).AddTokenAuth(token2)
		resp2 := MakeRequest(t, req2, http.StatusCreated)
		var fileResponse2 api.FileResponse
		DecodeJSON(t, resp2, &fileResponse2)
		assert.Equal(t, fileResponse2.Content.Path, treePath2)
		assert.EqualValues(t, 31, fileResponse2.Content.Size)

		createFileOptions3 := getCreateOptionsFile3()
		treePath3 := "dir2/dir3/file3.txt"
		req3 := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user2.Name, repo1.Name, treePath3), &createFileOptions3).AddTokenAuth(token2)
		resp3 := MakeRequest(t, req3, http.StatusCreated)
		var fileResponse3 api.FileResponse
		DecodeJSON(t, resp3, &fileResponse3)
		assert.Equal(t, fileResponse3.Content.Path, treePath3)
		assert.EqualValues(t, 36, fileResponse3.Content.Size)

		createFileOptions4 := getCreateOptionsFile4()
		treePath4 := "dir2/dir4/file4.txt"
		req4 := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user2.Name, repo1.Name, treePath4), &createFileOptions4).AddTokenAuth(token2)
		resp4 := MakeRequest(t, req4, http.StatusCreated)
		var fileResponse4 api.FileResponse
		DecodeJSON(t, resp4, &fileResponse4)
		assert.Equal(t, fileResponse4.Content.Path, treePath4)
		assert.EqualValues(t, 36, fileResponse4.Content.Size)

		// Verify file1 exists
		getReq1 := NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user2.Name, repo1.Name, treePath1))
		MakeRequest(t, getReq1, http.StatusOK)

		// Verify file2 exists
		getReq2 := NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user2.Name, repo1.Name, treePath2))
		MakeRequest(t, getReq2, http.StatusOK)

		// Verify file3 exists
		getReq3 := NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user2.Name, repo1.Name, treePath3))
		MakeRequest(t, getReq3, http.StatusOK)

		// Verify file4 exists
		getReq4 := NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user2.Name, repo1.Name, treePath4))
		MakeRequest(t, getReq4, http.StatusOK)

		treePathDirDel := ""
		req := NewRequest(t, "GET", fmt.Sprintf("/%s/%s/_delete_path/master/%s", user2.Name, repo1.Name, treePathDirDel))
		session.MakeRequest(t, req, http.StatusOK)

		csrf := GetCSRF(t, session, fmt.Sprintf("/%s/%s/_delete_path/master/%s", user2.Name, repo1.Name, treePathDirDel))

		commitForm := map[string]string{
			"_csrf":          csrf,
			"commit_summary": "Delete",
			"commit_message": "",
			"commit_choice":  "direct",
			"commit_mail_id": "-1",
		}

		// POST to the commit endpoint
		postReq := NewRequestWithValues(t, "POST", fmt.Sprintf("/%s/%s/_delete_path/master/%s", user2.Name, repo1.Name, treePathDirDel), commitForm)
		postResp := session.MakeRequest(t, postReq, http.StatusSeeOther)

		redirectLocation := postResp.Header().Get("Location")
		if redirectLocation != "" {
			verifyReq := NewRequest(t, "GET", redirectLocation)
			session.MakeRequest(t, verifyReq, http.StatusOK)
		}

		// Verify file1 exists
		getReq1 = NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user2.Name, repo1.Name, treePath1))
		MakeRequest(t, getReq1, http.StatusNotFound)

		// Verify file2 exists
		getReq2 = NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user2.Name, repo1.Name, treePath2))
		MakeRequest(t, getReq2, http.StatusNotFound)

		// Verify file3 exists
		getReq3 = NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user2.Name, repo1.Name, treePath3))
		MakeRequest(t, getReq3, http.StatusNotFound)

		// Verify file4 exists
		getReq4 = NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user2.Name, repo1.Name, treePath4))
		MakeRequest(t, getReq4, http.StatusNotFound)
	})
}

// Anonymous
func TestRecursiveDeleteAnonymous(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})       // owner of the repo1
		repo1 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1}) // public repo

		// Get user2's token
		session := loginUser(t, user2.Name)
		token2 := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)

		createFileOptions1 := getCreateOptionsFile1()
		treePath1 := "file1.txt"
		req1 := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user2.Name, repo1.Name, treePath1), &createFileOptions1).AddTokenAuth(token2)
		resp1 := MakeRequest(t, req1, http.StatusCreated)
		var fileResponse1 api.FileResponse
		DecodeJSON(t, resp1, &fileResponse1)
		assert.Equal(t, fileResponse1.Content.Path, treePath1)
		assert.EqualValues(t, 26, fileResponse1.Content.Size)

		createFileOptions2 := getCreateOptionsFile2()
		treePath2 := "dir2/file2.txt"
		req2 := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user2.Name, repo1.Name, treePath2), &createFileOptions2).AddTokenAuth(token2)
		resp2 := MakeRequest(t, req2, http.StatusCreated)
		var fileResponse2 api.FileResponse
		DecodeJSON(t, resp2, &fileResponse2)
		assert.Equal(t, fileResponse2.Content.Path, treePath2)
		assert.EqualValues(t, 31, fileResponse2.Content.Size)

		createFileOptions3 := getCreateOptionsFile3()
		treePath3 := "dir2/dir3/file3.txt"
		req3 := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user2.Name, repo1.Name, treePath3), &createFileOptions3).AddTokenAuth(token2)
		resp3 := MakeRequest(t, req3, http.StatusCreated)
		var fileResponse3 api.FileResponse
		DecodeJSON(t, resp3, &fileResponse3)
		assert.Equal(t, fileResponse3.Content.Path, treePath3)
		assert.EqualValues(t, 36, fileResponse3.Content.Size)

		createFileOptions4 := getCreateOptionsFile4()
		treePath4 := "dir2/dir4/file4.txt"
		req4 := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user2.Name, repo1.Name, treePath4), &createFileOptions4).AddTokenAuth(token2)
		resp4 := MakeRequest(t, req4, http.StatusCreated)
		var fileResponse4 api.FileResponse
		DecodeJSON(t, resp4, &fileResponse4)
		assert.Equal(t, fileResponse4.Content.Path, treePath4)
		assert.EqualValues(t, 36, fileResponse4.Content.Size)

		// Verify file1 exists
		getReq1 := NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user2.Name, repo1.Name, treePath1))
		MakeRequest(t, getReq1, http.StatusOK)

		// Verify file2 exists
		getReq2 := NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user2.Name, repo1.Name, treePath2))
		MakeRequest(t, getReq2, http.StatusOK)

		// Verify file3 exists
		getReq3 := NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user2.Name, repo1.Name, treePath3))
		MakeRequest(t, getReq3, http.StatusOK)

		// Verify file4 exists
		getReq4 := NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user2.Name, repo1.Name, treePath4))
		MakeRequest(t, getReq4, http.StatusOK)

		treePathDirDel := ""
		req := NewRequest(t, "GET", fmt.Sprintf("/%s/%s/_delete_path/master/%s", user2.Name, repo1.Name, treePathDirDel))
		MakeRequest(t, req, http.StatusSeeOther)
	})
}

// Other
func TestRecursiveDeleteOther(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})       // owner of the repo1
		user4 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 4})       // not the owner
		repo1 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1}) // public repo

		// Get user2's token
		session := loginUser(t, user2.Name)
		token2 := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)

		createFileOptions1 := getCreateOptionsFile1()
		treePath1 := "file1.txt"
		req1 := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user2.Name, repo1.Name, treePath1), &createFileOptions1).AddTokenAuth(token2)
		resp1 := MakeRequest(t, req1, http.StatusCreated)
		var fileResponse1 api.FileResponse
		DecodeJSON(t, resp1, &fileResponse1)
		assert.Equal(t, fileResponse1.Content.Path, treePath1)
		assert.EqualValues(t, 26, fileResponse1.Content.Size)

		createFileOptions2 := getCreateOptionsFile2()
		treePath2 := "dir2/file2.txt"
		req2 := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user2.Name, repo1.Name, treePath2), &createFileOptions2).AddTokenAuth(token2)
		resp2 := MakeRequest(t, req2, http.StatusCreated)
		var fileResponse2 api.FileResponse
		DecodeJSON(t, resp2, &fileResponse2)
		assert.Equal(t, fileResponse2.Content.Path, treePath2)
		assert.EqualValues(t, 31, fileResponse2.Content.Size)

		createFileOptions3 := getCreateOptionsFile3()
		treePath3 := "dir2/dir3/file3.txt"
		req3 := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user2.Name, repo1.Name, treePath3), &createFileOptions3).AddTokenAuth(token2)
		resp3 := MakeRequest(t, req3, http.StatusCreated)
		var fileResponse3 api.FileResponse
		DecodeJSON(t, resp3, &fileResponse3)
		assert.Equal(t, fileResponse3.Content.Path, treePath3)
		assert.EqualValues(t, 36, fileResponse3.Content.Size)

		createFileOptions4 := getCreateOptionsFile4()
		treePath4 := "dir2/dir4/file4.txt"
		req4 := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user2.Name, repo1.Name, treePath4), &createFileOptions4).AddTokenAuth(token2)
		resp4 := MakeRequest(t, req4, http.StatusCreated)
		var fileResponse4 api.FileResponse
		DecodeJSON(t, resp4, &fileResponse4)
		assert.Equal(t, fileResponse4.Content.Path, treePath4)
		assert.EqualValues(t, 36, fileResponse4.Content.Size)

		// Verify file1 exists
		getReq1 := NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user2.Name, repo1.Name, treePath1))
		MakeRequest(t, getReq1, http.StatusOK)

		// Verify file2 exists
		getReq2 := NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user2.Name, repo1.Name, treePath2))
		MakeRequest(t, getReq2, http.StatusOK)

		// Verify file3 exists
		getReq3 := NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user2.Name, repo1.Name, treePath3))
		MakeRequest(t, getReq3, http.StatusOK)

		// Verify file4 exists
		getReq4 := NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user2.Name, repo1.Name, treePath4))
		MakeRequest(t, getReq4, http.StatusOK)

		treePathDirDel := ""

		// user4
		session = loginUser(t, user4.Name)
		req := NewRequest(t, "GET", fmt.Sprintf("/%s/%s/_delete_path/master/%s", user2.Name, repo1.Name, treePathDirDel))
		session.MakeRequest(t, req, http.StatusNotFound)
	})
}

// Structure:
// file1.txt
// dir2/file2.txt
// dir2/dir3/file3.txt
// dir2/dir4/file4.txt
// delete: ROOT by colab user
func TestRecursiveDeleteRootColab(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
		user3 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 3})       // owner of the repo3
		repo3 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 3}) // public repo of user3

		// Get user2's token
		session := loginUser(t, user2.Name)
		token2 := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)

		createFileOptions1 := getCreateOptionsFile1()
		treePath1 := "file1.txt"
		req1 := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user3.Name, repo3.Name, treePath1), &createFileOptions1).AddTokenAuth(token2)
		resp1 := MakeRequest(t, req1, http.StatusCreated)
		var fileResponse1 api.FileResponse
		DecodeJSON(t, resp1, &fileResponse1)
		assert.Equal(t, fileResponse1.Content.Path, treePath1)
		assert.EqualValues(t, 26, fileResponse1.Content.Size)

		createFileOptions2 := getCreateOptionsFile2()
		treePath2 := "dir2/file2.txt"
		req2 := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user3.Name, repo3.Name, treePath2), &createFileOptions2).AddTokenAuth(token2)
		resp2 := MakeRequest(t, req2, http.StatusCreated)
		var fileResponse2 api.FileResponse
		DecodeJSON(t, resp2, &fileResponse2)
		assert.Equal(t, fileResponse2.Content.Path, treePath2)
		assert.EqualValues(t, 31, fileResponse2.Content.Size)

		createFileOptions3 := getCreateOptionsFile3()
		treePath3 := "dir2/dir3/file3.txt"
		req3 := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user3.Name, repo3.Name, treePath3), &createFileOptions3).AddTokenAuth(token2)
		resp3 := MakeRequest(t, req3, http.StatusCreated)
		var fileResponse3 api.FileResponse
		DecodeJSON(t, resp3, &fileResponse3)
		assert.Equal(t, fileResponse3.Content.Path, treePath3)
		assert.EqualValues(t, 36, fileResponse3.Content.Size)

		createFileOptions4 := getCreateOptionsFile4()
		treePath4 := "dir2/dir4/file4.txt"
		req4 := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user3.Name, repo3.Name, treePath4), &createFileOptions4).AddTokenAuth(token2)
		resp4 := MakeRequest(t, req4, http.StatusCreated)
		var fileResponse4 api.FileResponse
		DecodeJSON(t, resp4, &fileResponse4)
		assert.Equal(t, fileResponse4.Content.Path, treePath4)
		assert.EqualValues(t, 36, fileResponse4.Content.Size)

		// Verify file1 exists
		getReq1 := NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user3.Name, repo3.Name, treePath1)).AddTokenAuth(token2)
		MakeRequest(t, getReq1, http.StatusOK)

		// Verify file2 exists
		getReq2 := NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user3.Name, repo3.Name, treePath2)).AddTokenAuth(token2)
		MakeRequest(t, getReq2, http.StatusOK)

		// Verify file3 exists
		getReq3 := NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user3.Name, repo3.Name, treePath3)).AddTokenAuth(token2)
		MakeRequest(t, getReq3, http.StatusOK)

		// Verify file4 exists
		getReq4 := NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user3.Name, repo3.Name, treePath4)).AddTokenAuth(token2)
		MakeRequest(t, getReq4, http.StatusOK)

		treePathDirDel := ""
		req := NewRequest(t, "GET", fmt.Sprintf("/%s/%s/_delete_path/master/%s", user3.Name, repo3.Name, treePathDirDel))
		session.MakeRequest(t, req, http.StatusOK)

		csrf := GetCSRF(t, session, fmt.Sprintf("/%s/%s/_delete_path/master/%s", user3.Name, repo3.Name, treePathDirDel))

		commitForm := map[string]string{
			"_csrf":          csrf,
			"commit_summary": "Delete",
			"commit_message": "",
			"commit_choice":  "direct",
			"commit_mail_id": "-1",
		}

		// POST to the commit endpoint
		postReq := NewRequestWithValues(t, "POST", fmt.Sprintf("/%s/%s/_delete_path/master/%s", user3.Name, repo3.Name, treePathDirDel), commitForm)
		postResp := session.MakeRequest(t, postReq, http.StatusSeeOther)

		redirectLocation := postResp.Header().Get("Location")
		if redirectLocation != "" {
			verifyReq := NewRequest(t, "GET", redirectLocation)
			session.MakeRequest(t, verifyReq, http.StatusOK)
		}

		// Verify file1 exists
		getReq1 = NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user3.Name, repo3.Name, treePath1)).AddTokenAuth(token2)
		MakeRequest(t, getReq1, http.StatusNotFound)

		// Verify file2 exists
		getReq2 = NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user3.Name, repo3.Name, treePath2)).AddTokenAuth(token2)
		MakeRequest(t, getReq2, http.StatusNotFound)

		// Verify file3 exists
		getReq3 = NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user3.Name, repo3.Name, treePath3)).AddTokenAuth(token2)
		MakeRequest(t, getReq3, http.StatusNotFound)

		// Verify file4 exists
		getReq4 = NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user3.Name, repo3.Name, treePath4)).AddTokenAuth(token2)
		MakeRequest(t, getReq4, http.StatusNotFound)
	})
}
