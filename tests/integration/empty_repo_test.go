// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"testing"

	auth_model "forgejo.org/models/auth"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/json"
	"forgejo.org/modules/setting"
	api "forgejo.org/modules/structs"
	"forgejo.org/modules/test"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmptyRepo(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	subPaths := []string{
		"commits/master",
		"raw/foo",
		"commit/1ae57b34ccf7e18373",
		"graph",
	}
	emptyRepo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 6})
	assert.True(t, emptyRepo.IsEmpty)
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: emptyRepo.OwnerID})
	for _, subPath := range subPaths {
		req := NewRequestf(t, "GET", "/%s/%s/%s", owner.Name, emptyRepo.Name, subPath)
		MakeRequest(t, req, http.StatusNotFound)
	}
}

func TestEmptyRepoAddFile(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	session := loginUser(t, "user30")
	req := NewRequest(t, "GET", "/user30/empty/_new/"+setting.Repository.DefaultBranch)
	resp := session.MakeRequest(t, req, http.StatusOK)
	doc := NewHTMLParser(t, resp.Body).Find(`input[name="commit_choice"]`)
	assert.Empty(t, doc.AttrOr("checked", "_no_"))
	req = NewRequestWithValues(t, "POST", "/user30/empty/_new/"+setting.Repository.DefaultBranch, map[string]string{
		"_csrf":          GetCSRF(t, session, "/user/settings"),
		"commit_choice":  "direct",
		"tree_path":      "test-file.md",
		"content":        "newly-added-test-file",
		"commit_mail_id": "32",
	})

	resp = session.MakeRequest(t, req, http.StatusSeeOther)
	redirect := test.RedirectURL(resp)
	assert.Equal(t, "/user30/empty/src/branch/"+setting.Repository.DefaultBranch+"/test-file.md", redirect)

	req = NewRequest(t, "GET", redirect)
	resp = session.MakeRequest(t, req, http.StatusOK)
	assert.Contains(t, resp.Body.String(), "newly-added-test-file")
}

func TestEmptyRepoUploadFile(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	session := loginUser(t, "user30")
	req := NewRequest(t, "GET", "/user30/empty/_new/"+setting.Repository.DefaultBranch)
	resp := session.MakeRequest(t, req, http.StatusOK)
	doc := NewHTMLParser(t, resp.Body).Find(`input[name="commit_choice"]`)
	assert.Empty(t, doc.AttrOr("checked", "_no_"))

	body := &bytes.Buffer{}
	mpForm := multipart.NewWriter(body)
	_ = mpForm.WriteField("_csrf", GetCSRF(t, session, "/user/settings"))
	file, _ := mpForm.CreateFormFile("file", "uploaded-file.txt")
	_, _ = io.Copy(file, bytes.NewBufferString("newly-uploaded-test-file"))
	_ = mpForm.Close()

	req = NewRequestWithBody(t, "POST", "/user30/empty/upload-file", body)
	req.Header.Add("Content-Type", mpForm.FormDataContentType())
	resp = session.MakeRequest(t, req, http.StatusOK)
	respMap := map[string]string{}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &respMap))
	filesFullpathKey := fmt.Sprintf("files_fullpath[%s]", respMap["uuid"])
	req = NewRequestWithValues(t, "POST", "/user30/empty/_upload/"+setting.Repository.DefaultBranch, map[string]string{
		"_csrf":          GetCSRF(t, session, "/user/settings"),
		"commit_choice":  "direct",
		"files":          respMap["uuid"],
		filesFullpathKey: "uploaded-file.txt",
		"tree_path":      "",
		"commit_mail_id": "-1",
	})
	resp = session.MakeRequest(t, req, http.StatusSeeOther)
	redirect := test.RedirectURL(resp)
	assert.Equal(t, "/user30/empty/src/branch/"+setting.Repository.DefaultBranch+"/", redirect)

	req = NewRequest(t, "GET", redirect)
	resp = session.MakeRequest(t, req, http.StatusOK)
	assert.Contains(t, resp.Body.String(), "uploaded-file.txt")
}

func TestEmptyRepoAddFileByAPI(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	session := loginUser(t, "user30")
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository)

	req := NewRequestWithJSON(t, "POST", "/api/v1/repos/user30/empty/contents/new-file.txt", &api.CreateFileOptions{
		FileOptions: api.FileOptions{
			NewBranchName: "new_branch",
			Message:       "init",
		},
		ContentBase64: base64.StdEncoding.EncodeToString([]byte("newly-added-api-file")),
	}).AddTokenAuth(token)

	resp := MakeRequest(t, req, http.StatusCreated)
	var fileResponse api.FileResponse
	DecodeJSON(t, resp, &fileResponse)
	expectedHTMLURL := setting.AppURL + "user30/empty/src/branch/new_branch/new-file.txt"
	assert.Equal(t, expectedHTMLURL, *fileResponse.Content.HTMLURL)

	req = NewRequest(t, "GET", "/user30/empty/src/branch/new_branch/new-file.txt")
	resp = session.MakeRequest(t, req, http.StatusOK)
	assert.Contains(t, resp.Body.String(), "newly-added-api-file")

	req = NewRequest(t, "GET", "/api/v1/repos/user30/empty").
		AddTokenAuth(token)
	resp = session.MakeRequest(t, req, http.StatusOK)
	var apiRepo api.Repository
	DecodeJSON(t, resp, &apiRepo)
	assert.Equal(t, "new_branch", apiRepo.DefaultBranch)
}

func TestEmptyRepoAPIRequestsReturn404(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	session := loginUser(t, "user30")
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeReadRepository)

	t.Run("Raw", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, "GET", "/api/v1/repos/user30/empty/raw/main/something").AddTokenAuth(token)
		_ = session.MakeRequest(t, req, http.StatusNotFound)
	})

	t.Run("Media", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, "GET", "/api/v1/repos/user30/empty/media/main/something").AddTokenAuth(token)
		_ = session.MakeRequest(t, req, http.StatusNotFound)
	})
}
