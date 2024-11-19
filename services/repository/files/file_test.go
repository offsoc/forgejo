// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package files

import (
	"testing"

	"code.gitea.io/gitea/models/db"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/unittest"
	"code.gitea.io/gitea/modules/gitrepo"
	"code.gitea.io/gitea/modules/setting"
	api "code.gitea.io/gitea/modules/structs"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCleanUploadFileName(t *testing.T) {
	t.Run("Clean regular file", func(t *testing.T) {
		name := "this/is/test"
		cleanName := CleanUploadFileName(name)
		expectedCleanName := name
		assert.EqualValues(t, expectedCleanName, cleanName)
	})

	t.Run("Clean a .git path", func(t *testing.T) {
		name := "this/is/test/.git"
		cleanName := CleanUploadFileName(name)
		expectedCleanName := ""
		assert.EqualValues(t, expectedCleanName, cleanName)
	})
}

func getExpectedFileResponse() *api.FileResponse {
	treePath := "README.md"
	sha := "4b4851ad51df6a7d9f25c979345979eaeb5b349f"
	encoding := "base64"
	content := "IyByZXBvMQoKRGVzY3JpcHRpb24gZm9yIHJlcG8x"
	selfURL := setting.AppURL + "api/v1/repos/user2/repo1/contents/" + treePath + "?ref=master"
	htmlURL := setting.AppURL + "user2/repo1/src/branch/master/" + treePath
	gitURL := setting.AppURL + "api/v1/repos/user2/repo1/git/blobs/" + sha
	downloadURL := setting.AppURL + "user2/repo1/raw/branch/master/" + treePath
	return &api.FileResponse{
		Content: &api.ContentsResponse{
			Name:          treePath,
			Path:          treePath,
			SHA:           sha,
			LastCommitSHA: "65f1bf27bc3bf70f64657658635e66094edbcb4d",
			Type:          "file",
			Size:          30,
			Encoding:      &encoding,
			Content:       &content,
			URL:           &selfURL,
			HTMLURL:       &htmlURL,
			GitURL:        &gitURL,
			DownloadURL:   &downloadURL,
			Links: &api.FileLinksResponse{
				Self:    &selfURL,
				GitURL:  &gitURL,
				HTMLURL: &htmlURL,
			},
		},
		Commit: &api.FileCommitResponse{
			CommitMeta: api.CommitMeta{
				URL: "https://try.gitea.io/api/v1/repos/user2/repo1/git/commits/65f1bf27bc3bf70f64657658635e66094edbcb4d",
				SHA: "65f1bf27bc3bf70f64657658635e66094edbcb4d",
			},
			HTMLURL: "https://try.gitea.io/user2/repo1/commit/65f1bf27bc3bf70f64657658635e66094edbcb4d",
			Author: &api.CommitUser{
				Identity: api.Identity{
					Name:  "user1",
					Email: "address1@example.com",
				},
				Date: "2017-03-19T20:47:59Z",
			},
			Committer: &api.CommitUser{
				Identity: api.Identity{
					Name:  "Ethan Koenig",
					Email: "ethantkoenig@gmail.com",
				},
				Date: "2017-03-19T20:47:59Z",
			},
			Parents: []*api.CommitMeta{},
			Message: "Initial commit\n",
			Tree: &api.CommitMeta{
				URL: "https://try.gitea.io/api/v1/repos/user2/repo1/git/trees/2a2f1d4670728a2e10049e345bd7a276468beab6",
				SHA: "2a2f1d4670728a2e10049e345bd7a276468beab6",
			},
		},
		Verification: &api.PayloadCommitVerification{
			Verified:  false,
			Reason:    "gpg.error.not_signed_commit",
			Signature: "",
			Payload:   "",
		},
	}
}

func TestGetFileResponseFromCommit(t *testing.T) {
	unittest.PrepareTestEnv(t)

	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	branch := repo.DefaultBranch
	treePath := "README.md"
	gitRepo, _ := gitrepo.OpenRepository(db.DefaultContext, repo)
	defer gitRepo.Close()
	commit, _ := gitRepo.GetBranchCommit(branch)
	expectedFileResponse := getExpectedFileResponse()

	fileResponse, err := GetFileResponseFromCommit(db.DefaultContext, repo, commit, branch, treePath)
	require.NoError(t, err)
	assert.EqualValues(t, expectedFileResponse, fileResponse)
}
