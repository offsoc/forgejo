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

func TestDeleteNotPassedAssignee(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Fake issue with assignees
	issue, err := issues_model.GetIssueByID(db.DefaultContext, 1)
	require.NoError(t, err)

	err = issue.LoadAttributes(db.DefaultContext)
	require.NoError(t, err)

	assert.Len(t, issue.Assignees, 1)

	user1, err := user_model.GetUserByID(db.DefaultContext, 1) // This user is already assigned (see the definition in fixtures), so running  UpdateAssignee should unassign him
	require.NoError(t, err)

	// Check if he got removed
	isAssigned, err := issues_model.IsUserAssignedToIssue(db.DefaultContext, issue, user1)
	require.NoError(t, err)
	assert.True(t, isAssigned)

	// Clean everyone
	err = DeleteNotPassedAssignee(db.DefaultContext, issue, user1, []*user_model.User{})
	require.NoError(t, err)
	assert.Empty(t, issue.Assignees)

	// Reload to check they're gone
	issue.ResetAttributesLoaded()
	require.NoError(t, issue.LoadAssignees(db.DefaultContext))
	assert.Empty(t, issue.Assignees)
	assert.Empty(t, issue.Assignee)
}
