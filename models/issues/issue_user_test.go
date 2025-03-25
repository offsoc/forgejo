// Copyright 2017 The Gogs Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issues_test

import (
	"testing"

	"forgejo.org/models/db"
	issues_model "forgejo.org/models/issues"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"

	"github.com/stretchr/testify/require"
)

func Test_NewIssueUsers(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	newIssue := &issues_model.Issue{
		RepoID:   repo.ID,
		PosterID: 4,
		Index:    6,
		Title:    "newTestIssueTitle",
		Content:  "newTestIssueContent",
	}

	// artificially insert new issue
	unittest.AssertSuccessfulInsert(t, newIssue)

	require.NoError(t, issues_model.NewIssueUsers(db.DefaultContext, repo, newIssue))

	// issue_user table should now have entries for new issue
	unittest.AssertExistsAndLoadBean(t, &issues_model.IssueUser{IssueID: newIssue.ID, UID: newIssue.PosterID})
	unittest.AssertExistsAndLoadBean(t, &issues_model.IssueUser{IssueID: newIssue.ID, UID: repo.OwnerID})
}

func TestUpdateIssueUserByRead(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 1})

	require.NoError(t, issues_model.UpdateIssueUserByRead(db.DefaultContext, 4, issue.ID))
	unittest.AssertExistsAndLoadBean(t, &issues_model.IssueUser{IssueID: issue.ID, UID: 4}, "is_read=1")

	require.NoError(t, issues_model.UpdateIssueUserByRead(db.DefaultContext, 4, issue.ID))
	unittest.AssertExistsAndLoadBean(t, &issues_model.IssueUser{IssueID: issue.ID, UID: 4}, "is_read=1")

	require.NoError(t, issues_model.UpdateIssueUserByRead(db.DefaultContext, unittest.NonexistentID, unittest.NonexistentID))
}

func TestUpdateIssueUsersByMentions(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 1})

	uids := []int64{2, 5}
	require.NoError(t, issues_model.UpdateIssueUsersByMentions(db.DefaultContext, issue.ID, uids))
	for _, uid := range uids {
		unittest.AssertExistsAndLoadBean(t, &issues_model.IssueUser{IssueID: issue.ID, UID: uid}, "is_mentioned=1")
	}
}
