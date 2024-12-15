// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo_test

import (
	"testing"

	"code.gitea.io/gitea/models/db"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/unittest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetUserFork(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// User13 has repo 11 forked from repo10
	repo10, err := repo_model.GetRepositoryByID(db.DefaultContext, 10)
	require.NoError(t, err)
	assert.NotNil(t, repo10)
	repo11, err := repo_model.GetUserFork(db.DefaultContext, repo10, 13)
	require.NoError(t, err)
	assert.NotNil(t, repo11)
	assert.Equal(t, int64(11), repo11.ID)
	assert.Equal(t, int64(10), repo11.ForkID)

	// user13 does not have a fork of repo9
	repo9, err := repo_model.GetRepositoryByID(db.DefaultContext, 9)
	require.NoError(t, err)
	assert.NotNil(t, repo9)
	fork, err := repo_model.GetUserFork(db.DefaultContext, repo9, 13)
	require.NoError(t, err)
	assert.Nil(t, fork)

	// User15 has repo id 63 forked from repo10, which counts as a fork of repo11 since they have a common base
	fork, err = repo_model.GetUserFork(db.DefaultContext, repo11, 15)
	require.NoError(t, err)
	assert.NotNil(t, fork)
	assert.Equal(t, int64(63), fork.ID)
	assert.Equal(t, int64(10), fork.ForkID)
}

func TestHasGetForkedRepo(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// User13 has repo 11 forked from repo10
	repo10, err := repo_model.GetRepositoryByID(db.DefaultContext, 10)
	require.NoError(t, err)
	assert.NotNil(t, repo10)
	hasFork := repo_model.HasForkedRepo(db.DefaultContext, 13, repo10)
	require.True(t, hasFork)
	repo11 := repo_model.GetForkedRepo(db.DefaultContext, 13, repo10)
	assert.NotNil(t, repo11)
	assert.Equal(t, int64(11), repo11.ID)
	assert.Equal(t, int64(10), repo11.ForkID)

	// user13 does not have a fork of repo9
	repo9, err := repo_model.GetRepositoryByID(db.DefaultContext, 9)
	require.NoError(t, err)
	assert.NotNil(t, repo9)
	hasFork = repo_model.HasForkedRepo(db.DefaultContext, 13, repo9)
	require.False(t, hasFork)
	fork := repo_model.GetForkedRepo(db.DefaultContext, 13, repo9)
	require.NoError(t, err)
	assert.Nil(t, fork)

	// User15 has repo id 63 forked from repo10, which counts as a fork of repo11 since they have a common base
	hasFork = repo_model.HasForkedRepo(db.DefaultContext, 15, repo11)
	require.True(t, hasFork)
	fork = repo_model.GetForkedRepo(db.DefaultContext, 15, repo11)
	require.NoError(t, err)
	assert.NotNil(t, fork)
	assert.Equal(t, int64(63), fork.ID)
	assert.Equal(t, int64(10), fork.ForkID)
}
