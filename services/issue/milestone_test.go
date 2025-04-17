// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issue

import (
	"testing"

	"forgejo.org/models/db"
	issues_model "forgejo.org/models/issues"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChangeMilestoneAssign(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{RepoID: 1})
	doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	assert.NotNil(t, issue)
	assert.NotNil(t, doer)

	oldMilestoneID := issue.MilestoneID
	issue.MilestoneID = 2
	require.NoError(t, issue.LoadMilestone(db.DefaultContext))
	require.NoError(t, ChangeMilestoneAssign(db.DefaultContext, issue, doer, oldMilestoneID))
	unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{
		IssueID:        issue.ID,
		Type:           issues_model.CommentTypeMilestone,
		MilestoneID:    issue.MilestoneID,
		OldMilestoneID: oldMilestoneID,
	})
	unittest.CheckConsistencyFor(t, &issues_model.Milestone{}, &issues_model.Issue{})
	assert.NotNil(t, issue.Milestone)

	oldMilestoneID = issue.MilestoneID
	issue.MilestoneID = 0
	require.NoError(t, ChangeMilestoneAssign(db.DefaultContext, issue, doer, oldMilestoneID))
	assert.EqualValues(t, 0, issue.MilestoneID)
	assert.Nil(t, issue.Milestone)
}
