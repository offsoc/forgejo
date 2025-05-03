// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package files

import (
	"testing"
	"time"

	"forgejo.org/models/db"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	"forgejo.org/modules/gitrepo"
	api "forgejo.org/modules/structs"

	_ "forgejo.org/models/actions"
	_ "forgejo.org/models/forgefed"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	unittest.MainTest(m)
}

func getExpectedReadmeContentsResponse() *api.ContentsResponse {
	treePath := "README.md"
	sha := "4b4851ad51df6a7d9f25c979345979eaeb5b349f"
	encoding := "base64"
	content := "IyByZXBvMQoKRGVzY3JpcHRpb24gZm9yIHJlcG8x"
	selfURL := "https://try.gitea.io/api/v1/repos/user2/repo1/contents/" + treePath + "?ref=master"
	htmlURL := "https://try.gitea.io/user2/repo1/src/branch/master/" + treePath
	gitURL := "https://try.gitea.io/api/v1/repos/user2/repo1/git/blobs/" + sha
	downloadURL := "https://try.gitea.io/user2/repo1/raw/branch/master/" + treePath
	return &api.ContentsResponse{
		Name:           treePath,
		Path:           treePath,
		SHA:            "4b4851ad51df6a7d9f25c979345979eaeb5b349f",
		LastCommitSHA:  "65f1bf27bc3bf70f64657658635e66094edbcb4d",
		LastCommitWhen: time.Date(2017, time.March, 19, 16, 47, 59, 0, time.FixedZone("", -14400)),
		Type:           "file",
		Size:           30,
		Encoding:       &encoding,
		Content:        &content,
		URL:            &selfURL,
		HTMLURL:        &htmlURL,
		GitURL:         &gitURL,
		DownloadURL:    &downloadURL,
		Links: &api.FileLinksResponse{
			Self:    &selfURL,
			GitURL:  &gitURL,
			HTMLURL: &htmlURL,
		},
	}
}

func TestGetContents(t *testing.T) {
	unittest.PrepareTestEnv(t)
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})

	treePath := "README.md"
	ref := repo.DefaultBranch

	expectedContentsResponse := getExpectedReadmeContentsResponse()

	t.Run("Get README.md contents with GetContents(ctx, )", func(t *testing.T) {
		fileContentResponse, err := GetContents(db.DefaultContext, repo, treePath, ref, false)
		assert.Equal(t, expectedContentsResponse, fileContentResponse)
		require.NoError(t, err)
	})

	t.Run("Get README.md contents with ref as empty string (should then use the repo's default branch) with GetContents(ctx, )", func(t *testing.T) {
		fileContentResponse, err := GetContents(db.DefaultContext, repo, treePath, "", false)
		assert.Equal(t, expectedContentsResponse, fileContentResponse)
		require.NoError(t, err)
	})
}

func TestGetContentsOrListForDir(t *testing.T) {
	unittest.PrepareTestEnv(t)
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})

	treePath := "" // root dir
	ref := repo.DefaultBranch

	readmeContentsResponse := getExpectedReadmeContentsResponse()
	// because will be in a list, doesn't have encoding and content
	readmeContentsResponse.Encoding = nil
	readmeContentsResponse.Content = nil

	expectedContentsListResponse := []*api.ContentsResponse{
		readmeContentsResponse,
	}

	t.Run("Get root dir contents with GetContentsOrList(ctx, )", func(t *testing.T) {
		fileContentResponse, err := GetContentsOrList(db.DefaultContext, repo, treePath, ref)
		assert.EqualValues(t, expectedContentsListResponse, fileContentResponse)
		require.NoError(t, err)
	})

	t.Run("Get root dir contents with ref as empty string (should then use the repo's default branch) with GetContentsOrList(ctx, )", func(t *testing.T) {
		fileContentResponse, err := GetContentsOrList(db.DefaultContext, repo, treePath, "")
		assert.EqualValues(t, expectedContentsListResponse, fileContentResponse)
		require.NoError(t, err)
	})
}

func TestGetContentsOrListForFile(t *testing.T) {
	unittest.PrepareTestEnv(t)
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})

	treePath := "README.md"
	ref := repo.DefaultBranch

	expectedContentsResponse := getExpectedReadmeContentsResponse()

	t.Run("Get README.md contents with GetContentsOrList(ctx, )", func(t *testing.T) {
		fileContentResponse, err := GetContentsOrList(db.DefaultContext, repo, treePath, ref)
		assert.EqualValues(t, expectedContentsResponse, fileContentResponse)
		require.NoError(t, err)
	})

	t.Run("Get README.md contents with ref as empty string (should then use the repo's default branch) with GetContentsOrList(ctx, )", func(t *testing.T) {
		fileContentResponse, err := GetContentsOrList(db.DefaultContext, repo, treePath, "")
		assert.EqualValues(t, expectedContentsResponse, fileContentResponse)
		require.NoError(t, err)
	})
}

func TestGetContentsErrors(t *testing.T) {
	unittest.PrepareTestEnv(t)
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})

	treePath := "README.md"
	ref := repo.DefaultBranch

	t.Run("bad treePath", func(t *testing.T) {
		badTreePath := "bad/tree.md"
		fileContentResponse, err := GetContents(db.DefaultContext, repo, badTreePath, ref, false)
		require.EqualError(t, err, "object does not exist [id: , rel_path: bad]")
		assert.Nil(t, fileContentResponse)
	})

	t.Run("bad ref", func(t *testing.T) {
		badRef := "bad_ref"
		fileContentResponse, err := GetContents(db.DefaultContext, repo, treePath, badRef, false)
		require.EqualError(t, err, "object does not exist [id: "+badRef+", rel_path: ]")
		assert.Nil(t, fileContentResponse)
	})
}

func TestGetContentsOrListErrors(t *testing.T) {
	unittest.PrepareTestEnv(t)
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})

	treePath := "README.md"
	ref := repo.DefaultBranch

	t.Run("bad treePath", func(t *testing.T) {
		badTreePath := "bad/tree.md"
		fileContentResponse, err := GetContentsOrList(db.DefaultContext, repo, badTreePath, ref)
		require.EqualError(t, err, "object does not exist [id: , rel_path: bad]")
		assert.Nil(t, fileContentResponse)
	})

	t.Run("bad ref", func(t *testing.T) {
		badRef := "bad_ref"
		fileContentResponse, err := GetContentsOrList(db.DefaultContext, repo, treePath, badRef)
		require.EqualError(t, err, "object does not exist [id: "+badRef+", rel_path: ]")
		assert.Nil(t, fileContentResponse)
	})
}

func TestGetContentsOrListOfEmptyRepos(t *testing.T) {
	unittest.PrepareTestEnv(t)
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 52})

	t.Run("empty repo", func(t *testing.T) {
		contents, err := GetContentsOrList(db.DefaultContext, repo, "", "")
		require.NoError(t, err)
		assert.Empty(t, contents)
	})
}

func TestGetBlobBySHA(t *testing.T) {
	unittest.PrepareTestEnv(t)
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})

	gitRepo, err := gitrepo.OpenRepository(db.DefaultContext, repo)
	require.NoError(t, err)
	defer gitRepo.Close()

	gbr, err := GetBlobBySHA(db.DefaultContext, repo, gitRepo, "65f1bf27bc3bf70f64657658635e66094edbcb4d")
	expectedGBR := &api.GitBlobResponse{
		Content:  "dHJlZSAyYTJmMWQ0NjcwNzI4YTJlMTAwNDllMzQ1YmQ3YTI3NjQ2OGJlYWI2CmF1dGhvciB1c2VyMSA8YWRkcmVzczFAZXhhbXBsZS5jb20+IDE0ODk5NTY0NzkgLTA0MDAKY29tbWl0dGVyIEV0aGFuIEtvZW5pZyA8ZXRoYW50a29lbmlnQGdtYWlsLmNvbT4gMTQ4OTk1NjQ3OSAtMDQwMAoKSW5pdGlhbCBjb21taXQK",
		Encoding: "base64",
		URL:      "https://try.gitea.io/api/v1/repos/user2/repo1/git/blobs/65f1bf27bc3bf70f64657658635e66094edbcb4d",
		SHA:      "65f1bf27bc3bf70f64657658635e66094edbcb4d",
		Size:     180,
	}
	require.NoError(t, err)
	assert.Equal(t, expectedGBR, gbr)
}
