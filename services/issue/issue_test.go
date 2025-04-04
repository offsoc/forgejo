// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issue

import (
	"testing"

	"forgejo.org/models/db"
	issues_model "forgejo.org/models/issues"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetRefEndNamesAndURLs(t *testing.T) {
	issues := []*issues_model.Issue{
		{ID: 1, Ref: "refs/heads/branch1"},
		{ID: 2, Ref: "refs/tags/tag1"},
		{ID: 3, Ref: "c0ffee"},
	}
	repoLink := "/foo/bar"

	endNames, urls := GetRefEndNamesAndURLs(issues, repoLink)
	assert.Equal(t, map[int64]string{1: "branch1", 2: "tag1", 3: "c0ffee"}, endNames)
	assert.Equal(t, map[int64]string{
		1: repoLink + "/src/branch/branch1",
		2: repoLink + "/src/tag/tag1",
		3: repoLink + "/src/commit/c0ffee",
	}, urls)
}

func TestIssue_DeleteIssue(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	issueIDs, err := issues_model.GetIssueIDsByRepoID(db.DefaultContext, 1)
	require.NoError(t, err)
	assert.Len(t, issueIDs, 5)

	issue := &issues_model.Issue{
		RepoID: 1,
		ID:     issueIDs[2],
	}

	err = deleteIssue(db.DefaultContext, issue)
	require.NoError(t, err)
	issueIDs, err = issues_model.GetIssueIDsByRepoID(db.DefaultContext, 1)
	require.NoError(t, err)
	assert.Len(t, issueIDs, 4)

	// check attachment removal
	attachments, err := repo_model.GetAttachmentsByIssueID(db.DefaultContext, 4)
	require.NoError(t, err)
	issue, err = issues_model.GetIssueByID(db.DefaultContext, 4)
	require.NoError(t, err)
	err = deleteIssue(db.DefaultContext, issue)
	require.NoError(t, err)
	assert.Len(t, attachments, 2)
	for i := range attachments {
		attachment, err := repo_model.GetAttachmentByUUID(db.DefaultContext, attachments[i].UUID)
		require.Error(t, err)
		assert.True(t, repo_model.IsErrAttachmentNotExist(err))
		assert.Nil(t, attachment)
	}

	// check issue dependencies
	user, err := user_model.GetUserByID(db.DefaultContext, 1)
	require.NoError(t, err)
	issue1, err := issues_model.GetIssueByID(db.DefaultContext, 1)
	require.NoError(t, err)
	issue2, err := issues_model.GetIssueByID(db.DefaultContext, 2)
	require.NoError(t, err)
	err = issues_model.CreateIssueDependency(db.DefaultContext, user, issue1, issue2)
	require.NoError(t, err)
	left, err := issues_model.IssueNoDependenciesLeft(db.DefaultContext, issue1)
	require.NoError(t, err)
	assert.False(t, left)

	err = deleteIssue(db.DefaultContext, issue2)
	require.NoError(t, err)
	left, err = issues_model.IssueNoDependenciesLeft(db.DefaultContext, issue1)
	require.NoError(t, err)
	assert.True(t, left)
}
