// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package org

import (
	"testing"

	"forgejo.org/models/db"
	"forgejo.org/models/organization"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"

	"github.com/stretchr/testify/require"
)

func TestTeam_AddRepository(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	testSuccess := func(teamID, repoID int64) {
		team := unittest.AssertExistsAndLoadBean(t, &organization.Team{ID: teamID})
		repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: repoID})
		require.NoError(t, TeamAddRepository(db.DefaultContext, team, repo))
		unittest.AssertExistsAndLoadBean(t, &organization.TeamRepo{TeamID: teamID, RepoID: repoID})
		unittest.CheckConsistencyFor(t, &organization.Team{ID: teamID}, &repo_model.Repository{ID: repoID})
	}
	testSuccess(2, 3)
	testSuccess(2, 5)

	team := unittest.AssertExistsAndLoadBean(t, &organization.Team{ID: 1})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	require.Error(t, TeamAddRepository(db.DefaultContext, team, repo))
	unittest.CheckConsistencyFor(t, &organization.Team{ID: 1}, &repo_model.Repository{ID: 1})
}
