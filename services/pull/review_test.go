// Copyright 2024 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package pull_test

import (
	"testing"

	"forgejo.org/models/db"
	issues_model "forgejo.org/models/issues"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	pull_service "forgejo.org/services/pull"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDismissReview(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	pull := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{})
	require.NoError(t, pull.LoadIssue(db.DefaultContext))
	issue := pull.Issue
	require.NoError(t, issue.LoadRepo(db.DefaultContext))
	reviewer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
	review, err := issues_model.CreateReview(db.DefaultContext, issues_model.CreateReviewOptions{
		Issue:    issue,
		Reviewer: reviewer,
		Type:     issues_model.ReviewTypeReject,
	})

	require.NoError(t, err)
	issue.IsClosed = true
	pull.HasMerged = false
	require.NoError(t, issues_model.UpdateIssueCols(db.DefaultContext, issue, "is_closed"))
	require.NoError(t, pull.UpdateCols(db.DefaultContext, "has_merged"))
	_, err = pull_service.DismissReview(db.DefaultContext, review.ID, issue.RepoID, "", &user_model.User{}, false, false)
	require.Error(t, err)
	assert.True(t, pull_service.IsErrDismissRequestOnClosedPR(err))

	pull.HasMerged = true
	pull.Issue.IsClosed = false
	require.NoError(t, issues_model.UpdateIssueCols(db.DefaultContext, issue, "is_closed"))
	require.NoError(t, pull.UpdateCols(db.DefaultContext, "has_merged"))
	_, err = pull_service.DismissReview(db.DefaultContext, review.ID, issue.RepoID, "", &user_model.User{}, false, false)
	require.Error(t, err)
	assert.True(t, pull_service.IsErrDismissRequestOnClosedPR(err))
}
