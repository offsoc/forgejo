// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repository

import (
	"testing"

	activities_model "forgejo.org/models/activities"
	"forgejo.org/models/db"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateRepositoryVisibilityChanged(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Get sample repo and change visibility
	repo, err := repo_model.GetRepositoryByID(db.DefaultContext, 9)
	require.NoError(t, err)
	repo.IsPrivate = true

	// Update it
	err = UpdateRepository(db.DefaultContext, repo, true)
	require.NoError(t, err)

	// Check visibility of action has become private
	act := activities_model.Action{}
	_, err = db.GetEngine(db.DefaultContext).ID(3).Get(&act)

	require.NoError(t, err)
	assert.True(t, act.IsPrivate)
}

func TestGetDirectorySize(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	repo, err := repo_model.GetRepositoryByID(db.DefaultContext, 1)
	require.NoError(t, err)

	size, err := getDirectorySize(repo.RepoPath())
	require.NoError(t, err)
	assert.Equal(t, size, repo.Size)
}
