// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repository

import (
	"testing"

	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/git"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestForkRepository(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// user 13 has already forked repo10
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 13})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 10})

	fork, err := ForkRepositoryAndUpdates(git.DefaultContext, user, user, ForkRepoOptions{
		BaseRepo:    repo,
		Name:        "test",
		Description: "test",
	})
	assert.Nil(t, fork)
	require.Error(t, err)
	assert.True(t, IsErrForkAlreadyExist(err))

	// user not reached maximum limit of repositories
	assert.False(t, repo_model.IsErrReachLimitOfRepo(err))

	// change AllowForkWithoutMaximumLimit to false for the test
	defer test.MockVariableValue(&setting.Repository.AllowForkWithoutMaximumLimit, false)()
	// user has reached maximum limit of repositories
	user.MaxRepoCreation = 0
	fork2, err := ForkRepositoryAndUpdates(git.DefaultContext, user, user, ForkRepoOptions{
		BaseRepo:    repo,
		Name:        "test",
		Description: "test",
	})
	assert.Nil(t, fork2)
	assert.True(t, repo_model.IsErrReachLimitOfRepo(err))
}
