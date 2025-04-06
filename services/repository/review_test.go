// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repository

import (
	"testing"

	"forgejo.org/models/db"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepoGetReviewerTeams(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	repo2 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 2})
	teams, err := GetReviewerTeams(db.DefaultContext, repo2)
	require.NoError(t, err)
	assert.Empty(t, teams)

	repo3 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 3})
	teams, err = GetReviewerTeams(db.DefaultContext, repo3)
	require.NoError(t, err)
	assert.Len(t, teams, 2)
}
