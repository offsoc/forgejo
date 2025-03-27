// Copyright 2018 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issues_test

import (
	"testing"

	"forgejo.org/models/db"
	issues_model "forgejo.org/models/issues"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateIssueDependency(t *testing.T) {
	// Prepare
	require.NoError(t, unittest.PrepareTestDatabase())

	user1, err := user_model.GetUserByID(db.DefaultContext, 1)
	require.NoError(t, err)

	issue1, err := issues_model.GetIssueByID(db.DefaultContext, 1)
	require.NoError(t, err)

	issue2, err := issues_model.GetIssueByID(db.DefaultContext, 2)
	require.NoError(t, err)

	// Create a dependency and check if it was successful
	err = issues_model.CreateIssueDependency(db.DefaultContext, user1, issue1, issue2)
	require.NoError(t, err)

	// Do it again to see if it will check if the dependency already exists
	err = issues_model.CreateIssueDependency(db.DefaultContext, user1, issue1, issue2)
	require.Error(t, err)
	assert.True(t, issues_model.IsErrDependencyExists(err))

	// Check for circular dependencies
	err = issues_model.CreateIssueDependency(db.DefaultContext, user1, issue2, issue1)
	require.Error(t, err)
	assert.True(t, issues_model.IsErrCircularDependency(err))

	_ = unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{Type: issues_model.CommentTypeAddDependency, PosterID: user1.ID, IssueID: issue1.ID})

	// Check if dependencies left is correct
	left, err := issues_model.IssueNoDependenciesLeft(db.DefaultContext, issue1)
	require.NoError(t, err)
	assert.False(t, left)

	// Close #2 and check again
	_, err = issues_model.ChangeIssueStatus(db.DefaultContext, issue2, user1, true)
	require.NoError(t, err)

	left, err = issues_model.IssueNoDependenciesLeft(db.DefaultContext, issue1)
	require.NoError(t, err)
	assert.True(t, left)

	// Test removing the dependency
	err = issues_model.RemoveIssueDependency(db.DefaultContext, user1, issue1, issue2, issues_model.DependencyTypeBlockedBy)
	require.NoError(t, err)
}
