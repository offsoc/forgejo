// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"testing"

	"forgejo.org/models/db"
	"forgejo.org/models/organization"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	webhook_model "forgejo.org/models/webhook"
	repo_service "forgejo.org/services/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTeam_HasRepository(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	test := func(teamID, repoID int64, expected bool) {
		team := unittest.AssertExistsAndLoadBean(t, &organization.Team{ID: teamID})
		assert.Equal(t, expected, repo_service.HasRepository(db.DefaultContext, team, repoID))
	}
	test(1, 1, false)
	test(1, 3, true)
	test(1, 5, true)
	test(1, unittest.NonexistentID, false)

	test(2, 3, true)
	test(2, 5, false)
}

func TestTeam_RemoveRepository(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	testSuccess := func(teamID, repoID int64) {
		team := unittest.AssertExistsAndLoadBean(t, &organization.Team{ID: teamID})
		require.NoError(t, repo_service.RemoveRepositoryFromTeam(db.DefaultContext, team, repoID))
		unittest.AssertNotExistsBean(t, &organization.TeamRepo{TeamID: teamID, RepoID: repoID})
		unittest.CheckConsistencyFor(t, &organization.Team{ID: teamID}, &repo_model.Repository{ID: repoID})
	}
	testSuccess(2, 3)
	testSuccess(2, 5)
	testSuccess(1, unittest.NonexistentID)
}

func TestDeleteOwnerRepositoriesDirectly(t *testing.T) {
	unittest.PrepareTestEnv(t)

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	deletedHookID := unittest.AssertExistsAndLoadBean(t, &webhook_model.Webhook{RepoID: 1}).ID
	unittest.AssertExistsAndLoadBean(t, &webhook_model.HookTask{
		HookID: deletedHookID,
	})

	preservedHookID := unittest.AssertExistsAndLoadBean(t, &webhook_model.Webhook{RepoID: 3}).ID
	unittest.AssertExistsAndLoadBean(t, &webhook_model.HookTask{
		HookID: preservedHookID,
	})

	require.NoError(t, repo_service.DeleteOwnerRepositoriesDirectly(db.DefaultContext, user))

	unittest.AssertNotExistsBean(t, &webhook_model.HookTask{
		HookID: deletedHookID,
	})
	unittest.AssertExistsAndLoadBean(t, &webhook_model.HookTask{
		HookID: preservedHookID,
	})
}
