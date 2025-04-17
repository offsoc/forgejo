// Copyright 2018 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package git

import (
	"path/filepath"
	"testing"

	"forgejo.org/modules/setting"
	"forgejo.org/modules/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepository_GetCommitBranches(t *testing.T) {
	bareRepo1Path := filepath.Join(testReposDir, "repo1_bare")
	bareRepo1, err := openRepositoryWithDefaultContext(bareRepo1Path)
	require.NoError(t, err)
	defer bareRepo1.Close()

	// these test case are specific to the repo1_bare test repo
	testCases := []struct {
		CommitID         string
		ExpectedBranches []string
	}{
		{"2839944139e0de9737a044f78b0e4b40d989a9e3", []string{"branch1"}},
		{"5c80b0245c1c6f8343fa418ec374b13b5d4ee658", []string{"branch2"}},
		{"37991dec2c8e592043f47155ce4808d4580f9123", []string{"master"}},
		{"95bb4d39648ee7e325106df01a621c530863a653", []string{"branch1", "branch2"}},
		{"8d92fc957a4d7cfd98bc375f0b7bb189a0d6c9f2", []string{"branch2", "master"}},
		{"master", []string{"master"}},
	}
	for _, testCase := range testCases {
		commit, err := bareRepo1.GetCommit(testCase.CommitID)
		require.NoError(t, err)
		branches, err := bareRepo1.getBranches(commit, 2)
		require.NoError(t, err)
		assert.Equal(t, testCase.ExpectedBranches, branches)
	}
}

func TestGetTagCommitWithSignature(t *testing.T) {
	bareRepo1Path := filepath.Join(testReposDir, "repo1_bare")
	bareRepo1, err := openRepositoryWithDefaultContext(bareRepo1Path)
	require.NoError(t, err)
	defer bareRepo1.Close()

	// both the tag and the commit are signed here, this validates only the commit signature
	commit, err := bareRepo1.GetCommit("28b55526e7100924d864dd89e35c1ea62e7a5a32")
	require.NoError(t, err)
	assert.NotNil(t, commit)
	assert.NotNil(t, commit.Signature)
	// test that signature is not in message
	assert.Equal(t, "signed-commit\n", commit.CommitMessage)
}

func TestGetCommitWithBadCommitID(t *testing.T) {
	bareRepo1Path := filepath.Join(testReposDir, "repo1_bare")
	bareRepo1, err := openRepositoryWithDefaultContext(bareRepo1Path)
	require.NoError(t, err)
	defer bareRepo1.Close()

	commit, err := bareRepo1.GetCommit("bad_branch")
	assert.Nil(t, commit)
	require.Error(t, err)
	assert.True(t, IsErrNotExist(err))
}

func TestIsCommitInBranch(t *testing.T) {
	bareRepo1Path := filepath.Join(testReposDir, "repo1_bare")
	bareRepo1, err := openRepositoryWithDefaultContext(bareRepo1Path)
	require.NoError(t, err)
	defer bareRepo1.Close()

	result, err := bareRepo1.IsCommitInBranch("2839944139e0de9737a044f78b0e4b40d989a9e3", "branch1")
	require.NoError(t, err)
	assert.True(t, result)

	result, err = bareRepo1.IsCommitInBranch("2839944139e0de9737a044f78b0e4b40d989a9e3", "branch2")
	require.NoError(t, err)
	assert.False(t, result)
}

func TestRepository_CommitsBetweenIDs(t *testing.T) {
	bareRepo1Path := filepath.Join(testReposDir, "repo4_commitsbetween")
	bareRepo1, err := openRepositoryWithDefaultContext(bareRepo1Path)
	require.NoError(t, err)
	defer bareRepo1.Close()

	cases := []struct {
		OldID           string
		NewID           string
		ExpectedCommits int
	}{
		{"fdc1b615bdcff0f0658b216df0c9209e5ecb7c78", "78a445db1eac62fe15e624e1137965969addf344", 1}, // com1 -> com2
		{"78a445db1eac62fe15e624e1137965969addf344", "fdc1b615bdcff0f0658b216df0c9209e5ecb7c78", 0}, // reset HEAD~, com2 -> com1
		{"78a445db1eac62fe15e624e1137965969addf344", "a78e5638b66ccfe7e1b4689d3d5684e42c97d7ca", 1}, // com2 -> com2_new
	}
	for i, c := range cases {
		commits, err := bareRepo1.CommitsBetweenIDs(c.NewID, c.OldID)
		require.NoError(t, err)
		assert.Len(t, commits, c.ExpectedCommits, "case %d", i)
	}
}

func TestGetTagCommit(t *testing.T) {
	t.Setenv("GIT_COMMITTER_DATE", "2006-01-01 13:37")
	bareRepo1Path := filepath.Join(testReposDir, "repo1_bare")

	clonedPath, err := cloneRepo(t, bareRepo1Path)
	require.NoError(t, err)

	bareRepo1, err := openRepositoryWithDefaultContext(clonedPath)
	require.NoError(t, err)
	defer bareRepo1.Close()

	lTagCommitID := "6fbd69e9823458e6c4a2fc5c0f6bc022b2f2acd1"
	lTagName := "lightweightTag"
	bareRepo1.CreateTag(lTagName, lTagCommitID)

	aTagCommitID := "8006ff9adbf0cb94da7dad9e537e53817f9fa5c0"
	aTagName := "annotatedTag"
	aTagMessage := "my annotated message"
	bareRepo1.CreateAnnotatedTag(aTagName, aTagMessage, aTagCommitID)

	aTagID, err := bareRepo1.GetTagCommitID(aTagName)
	require.NoError(t, err)
	assert.NotEqual(t, aTagCommitID, aTagID)

	lTagID, err := bareRepo1.GetTagCommitID(lTagName)
	require.NoError(t, err)
	assert.Equal(t, lTagCommitID, lTagID)

	aTag, err := bareRepo1.GetTagCommit(aTagName)
	require.NoError(t, err)
	assert.Equal(t, aTagCommitID, aTag.ID.String())

	lTag, err := bareRepo1.GetTagCommit(lTagName)
	require.NoError(t, err)
	assert.Equal(t, lTagCommitID, lTag.ID.String())
}

func TestCommitsByRange(t *testing.T) {
	bareRepo1Path := filepath.Join(testReposDir, "repo1_bare")
	bareRepo1, err := openRepositoryWithDefaultContext(bareRepo1Path)
	require.NoError(t, err)
	defer bareRepo1.Close()

	baseCommit, err := bareRepo1.GetBranchCommit("master")
	require.NoError(t, err)

	testCases := []struct {
		Page                int
		ExpectedCommitCount int
	}{
		{1, 3},
		{2, 3},
		{3, 1},
		{4, 0},
	}
	for _, testCase := range testCases {
		commits, err := baseCommit.CommitsByRange(testCase.Page, 3, "")
		require.NoError(t, err)
		assert.Len(t, commits, testCase.ExpectedCommitCount, "page: %d", testCase.Page)
	}
}

func TestCommitsByFileAndRange(t *testing.T) {
	bareRepo1Path := filepath.Join(testReposDir, "repo1_bare")
	bareRepo1, err := openRepositoryWithDefaultContext(bareRepo1Path)
	require.NoError(t, err)
	defer bareRepo1.Close()
	defer test.MockVariableValue(&setting.Git.CommitsRangeSize, 2)()

	testCases := []struct {
		File                string
		Page                int
		ExpectedCommitCount int
	}{
		{"file1.txt", 1, 1},
		{"file2.txt", 1, 1},
		{"file*.txt", 1, 2},
		{"foo", 1, 2},
		{"foo", 2, 1},
		{"foo", 3, 0},
		{"f*", 1, 2},
		{"f*", 2, 2},
		{"f*", 3, 1},
	}
	for _, testCase := range testCases {
		commits, err := bareRepo1.CommitsByFileAndRange(CommitsByFileAndRangeOptions{
			Revision: "master",
			File:     testCase.File,
			Page:     testCase.Page,
		})
		require.NoError(t, err)
		assert.Len(t, commits, testCase.ExpectedCommitCount, "file: '%s', page: %d", testCase.File, testCase.Page)
	}
}
