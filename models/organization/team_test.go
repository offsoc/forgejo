// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package organization_test

import (
	"path/filepath"
	"testing"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/organization"
	"code.gitea.io/gitea/models/perm"
	"code.gitea.io/gitea/models/unittest"
	"code.gitea.io/gitea/modules/setting"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTeam_IsOwnerTeam(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	team := unittest.AssertExistsAndLoadBean(t, &organization.Team{ID: 1})
	assert.True(t, team.IsOwnerTeam())

	team = unittest.AssertExistsAndLoadBean(t, &organization.Team{ID: 2})
	assert.False(t, team.IsOwnerTeam())
}

func TestTeam_IsMember(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	team := unittest.AssertExistsAndLoadBean(t, &organization.Team{ID: 1})
	assert.True(t, team.IsMember(db.DefaultContext, 2))
	assert.False(t, team.IsMember(db.DefaultContext, 4))
	assert.False(t, team.IsMember(db.DefaultContext, unittest.NonexistentID))

	team = unittest.AssertExistsAndLoadBean(t, &organization.Team{ID: 2})
	assert.True(t, team.IsMember(db.DefaultContext, 2))
	assert.True(t, team.IsMember(db.DefaultContext, 4))
	assert.False(t, team.IsMember(db.DefaultContext, unittest.NonexistentID))
}

func TestTeam_GetRepositories(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	test := func(teamID int64) {
		team := unittest.AssertExistsAndLoadBean(t, &organization.Team{ID: teamID})
		require.NoError(t, team.LoadRepositories(db.DefaultContext))
		assert.Len(t, team.Repos, team.NumRepos)
		for _, repo := range team.Repos {
			unittest.AssertExistsAndLoadBean(t, &organization.TeamRepo{TeamID: teamID, RepoID: repo.ID})
		}
	}
	test(1)
	test(3)
}

func TestTeam_GetMembers(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	test := func(teamID int64) {
		team := unittest.AssertExistsAndLoadBean(t, &organization.Team{ID: teamID})
		require.NoError(t, team.LoadMembers(db.DefaultContext))
		assert.Len(t, team.Members, team.NumMembers)
		for _, member := range team.Members {
			unittest.AssertExistsAndLoadBean(t, &organization.TeamUser{UID: member.ID, TeamID: teamID})
		}
	}
	test(1)
	test(3)
}

func TestGetTeam(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	testSuccess := func(orgID int64, name string) {
		team, err := organization.GetTeam(db.DefaultContext, orgID, name)
		require.NoError(t, err)
		assert.EqualValues(t, orgID, team.OrgID)
		assert.Equal(t, name, team.Name)
	}
	testSuccess(3, "Owners")
	testSuccess(3, "team1")

	_, err := organization.GetTeam(db.DefaultContext, 3, "nonexistent")
	require.Error(t, err)
	_, err = organization.GetTeam(db.DefaultContext, unittest.NonexistentID, "Owners")
	require.Error(t, err)
}

func TestGetTeamByID(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	testSuccess := func(teamID int64) {
		team, err := organization.GetTeamByID(db.DefaultContext, teamID)
		require.NoError(t, err)
		assert.EqualValues(t, teamID, team.ID)
	}
	testSuccess(1)
	testSuccess(2)
	testSuccess(3)
	testSuccess(4)

	_, err := organization.GetTeamByID(db.DefaultContext, unittest.NonexistentID)
	require.Error(t, err)
}

func TestIsTeamMember(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	test := func(orgID, teamID, userID int64, expected bool) {
		isMember, err := organization.IsTeamMember(db.DefaultContext, orgID, teamID, userID)
		require.NoError(t, err)
		assert.Equal(t, expected, isMember)
	}

	test(3, 1, 2, true)
	test(3, 1, 4, false)
	test(3, 1, unittest.NonexistentID, false)

	test(3, 2, 2, true)
	test(3, 2, 4, true)

	test(3, unittest.NonexistentID, unittest.NonexistentID, false)
	test(unittest.NonexistentID, unittest.NonexistentID, unittest.NonexistentID, false)
}

func TestGetTeamMembers(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	test := func(teamID int64) {
		team := unittest.AssertExistsAndLoadBean(t, &organization.Team{ID: teamID})
		members, err := organization.GetTeamMembers(db.DefaultContext, &organization.SearchMembersOptions{
			TeamID: teamID,
		})
		require.NoError(t, err)
		assert.Len(t, members, team.NumMembers)
		for _, member := range members {
			unittest.AssertExistsAndLoadBean(t, &organization.TeamUser{UID: member.ID, TeamID: teamID})
		}
	}
	test(1)
	test(3)
}

func TestGetUserTeams(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	test := func(userID int64) {
		teams, _, err := organization.SearchTeam(db.DefaultContext, &organization.SearchTeamOptions{UserID: userID})
		require.NoError(t, err)
		for _, team := range teams {
			unittest.AssertExistsAndLoadBean(t, &organization.TeamUser{TeamID: team.ID, UID: userID})
		}
	}
	test(2)
	test(5)
	test(unittest.NonexistentID)
}

func TestGetUserOrgTeams(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	test := func(orgID, userID int64) {
		teams, err := organization.GetUserOrgTeams(db.DefaultContext, orgID, userID)
		require.NoError(t, err)
		for _, team := range teams {
			assert.EqualValues(t, orgID, team.OrgID)
			unittest.AssertExistsAndLoadBean(t, &organization.TeamUser{TeamID: team.ID, UID: userID})
		}
	}
	test(3, 2)
	test(3, 4)
	test(3, unittest.NonexistentID)
}

func TestHasTeamRepo(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	test := func(teamID, repoID int64, expected bool) {
		team := unittest.AssertExistsAndLoadBean(t, &organization.Team{ID: teamID})
		assert.Equal(t, expected, organization.HasTeamRepo(db.DefaultContext, team.OrgID, teamID, repoID))
	}
	test(1, 1, false)
	test(1, 3, true)
	test(1, 5, true)
	test(1, unittest.NonexistentID, false)

	test(2, 3, true)
	test(2, 5, false)
}

func TestInconsistentOwnerTeam(t *testing.T) {
	defer unittest.OverrideFixtures(
		unittest.FixturesOptions{
			Dir:  filepath.Join(setting.AppWorkPath, "models/fixtures/"),
			Base: setting.AppWorkPath,
			Dirs: []string{"models/organization/TestInconsistentOwnerTeam/"},
		},
	)()
	require.NoError(t, unittest.PrepareTestDatabase())

	unittest.AssertExistsAndLoadBean(t, &organization.TeamUnit{ID: 1000, TeamID: 1000, AccessMode: perm.AccessModeNone})
	unittest.AssertExistsAndLoadBean(t, &organization.TeamUnit{ID: 1001, TeamID: 1000, AccessMode: perm.AccessModeNone})
	unittest.AssertExistsAndLoadBean(t, &organization.TeamUnit{ID: 1002, TeamID: 1000, AccessMode: perm.AccessModeNone})
	unittest.AssertExistsAndLoadBean(t, &organization.TeamUnit{ID: 1003, TeamID: 1000, AccessMode: perm.AccessModeNone})
	unittest.AssertExistsAndLoadBean(t, &organization.TeamUnit{ID: 1004, TeamID: 1000, AccessMode: perm.AccessModeNone})
	unittest.AssertExistsAndLoadBean(t, &organization.TeamUnit{ID: 1005, TeamID: 1000, AccessMode: perm.AccessModeNone})
	unittest.AssertExistsAndLoadBean(t, &organization.TeamUnit{ID: 1006, TeamID: 1000, AccessMode: perm.AccessModeNone})
	unittest.AssertExistsAndLoadBean(t, &organization.TeamUnit{ID: 1007, TeamID: 1000, AccessMode: perm.AccessModeNone})
	unittest.AssertExistsAndLoadBean(t, &organization.TeamUnit{ID: 1008, TeamID: 1000, AccessMode: perm.AccessModeNone})
	unittest.AssertExistsAndLoadBean(t, &organization.TeamUnit{ID: 1009, TeamID: 1000, AccessMode: perm.AccessModeNone})

	count, err := organization.CountInconsistentOwnerTeams(db.DefaultContext)
	require.NoError(t, err)
	require.EqualValues(t, 1, count)

	count, err = organization.FixInconsistentOwnerTeams(db.DefaultContext)
	require.NoError(t, err)
	require.EqualValues(t, 1, count)

	count, err = organization.CountInconsistentOwnerTeams(db.DefaultContext)
	require.NoError(t, err)
	require.EqualValues(t, 0, count)

	unittest.AssertExistsAndLoadBean(t, &organization.TeamUnit{ID: 1000, AccessMode: perm.AccessModeOwner})
	unittest.AssertExistsAndLoadBean(t, &organization.TeamUnit{ID: 1001, AccessMode: perm.AccessModeOwner})
	unittest.AssertExistsAndLoadBean(t, &organization.TeamUnit{ID: 1002, AccessMode: perm.AccessModeOwner})
	unittest.AssertExistsAndLoadBean(t, &organization.TeamUnit{ID: 1003, AccessMode: perm.AccessModeOwner})
	unittest.AssertExistsAndLoadBean(t, &organization.TeamUnit{ID: 1004, AccessMode: perm.AccessModeOwner})
	unittest.AssertExistsAndLoadBean(t, &organization.TeamUnit{ID: 1007, AccessMode: perm.AccessModeOwner})
	unittest.AssertExistsAndLoadBean(t, &organization.TeamUnit{ID: 1008, AccessMode: perm.AccessModeOwner})
	unittest.AssertExistsAndLoadBean(t, &organization.TeamUnit{ID: 1009, AccessMode: perm.AccessModeOwner})

	// External wiki and issue
	unittest.AssertExistsAndLoadBean(t, &organization.TeamUnit{ID: 1005, AccessMode: perm.AccessModeRead})
	unittest.AssertExistsAndLoadBean(t, &organization.TeamUnit{ID: 1006, AccessMode: perm.AccessModeRead})
}
