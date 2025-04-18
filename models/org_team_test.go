// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package models

import (
	"strings"
	"testing"

	"forgejo.org/models/db"
	"forgejo.org/models/organization"
	"forgejo.org/models/perm"
	access_model "forgejo.org/models/perm/access"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTeam_AddMember(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	test := func(teamID, userID int64) {
		team := unittest.AssertExistsAndLoadBean(t, &organization.Team{ID: teamID})
		require.NoError(t, AddTeamMember(db.DefaultContext, team, userID))
		unittest.AssertExistsAndLoadBean(t, &organization.TeamUser{UID: userID, TeamID: teamID})
		unittest.CheckConsistencyFor(t, &organization.Team{ID: teamID}, &user_model.User{ID: team.OrgID})
	}
	test(1, 2)
	test(1, 4)
	test(3, 2)
}

func TestTeam_RemoveMember(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	testSuccess := func(teamID, userID int64) {
		team := unittest.AssertExistsAndLoadBean(t, &organization.Team{ID: teamID})
		require.NoError(t, RemoveTeamMember(db.DefaultContext, team, userID))
		unittest.AssertNotExistsBean(t, &organization.TeamUser{UID: userID, TeamID: teamID})
		unittest.CheckConsistencyFor(t, &organization.Team{ID: teamID})
	}
	testSuccess(1, 4)
	testSuccess(2, 2)
	testSuccess(3, 2)
	testSuccess(3, unittest.NonexistentID)

	team := unittest.AssertExistsAndLoadBean(t, &organization.Team{ID: 1})
	err := RemoveTeamMember(db.DefaultContext, team, 2)
	assert.True(t, organization.IsErrLastOrgOwner(err))
}

func TestIsUsableTeamName(t *testing.T) {
	require.NoError(t, organization.IsUsableTeamName("usable"))
	assert.True(t, db.IsErrNameReserved(organization.IsUsableTeamName("new")))
}

func TestNewTeam(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	const teamName = "newTeamName"
	team := &organization.Team{Name: teamName, OrgID: 3}
	require.NoError(t, NewTeam(db.DefaultContext, team))
	unittest.AssertExistsAndLoadBean(t, &organization.Team{Name: teamName})
	unittest.CheckConsistencyFor(t, &organization.Team{}, &user_model.User{ID: team.OrgID})
}

func TestUpdateTeam(t *testing.T) {
	// successful update
	require.NoError(t, unittest.PrepareTestDatabase())

	team := unittest.AssertExistsAndLoadBean(t, &organization.Team{ID: 2})
	team.LowerName = "newname"
	team.Name = "newName"
	team.Description = strings.Repeat("A long description!", 100)
	team.AccessMode = perm.AccessModeAdmin
	require.NoError(t, UpdateTeam(db.DefaultContext, team, true, false))

	team = unittest.AssertExistsAndLoadBean(t, &organization.Team{Name: "newName"})
	assert.True(t, strings.HasPrefix(team.Description, "A long description!"))

	access := unittest.AssertExistsAndLoadBean(t, &access_model.Access{UserID: 4, RepoID: 3})
	assert.Equal(t, perm.AccessModeAdmin, access.Mode)

	unittest.CheckConsistencyFor(t, &organization.Team{ID: team.ID})
}

func TestUpdateTeam2(t *testing.T) {
	// update to already-existing team
	require.NoError(t, unittest.PrepareTestDatabase())

	team := unittest.AssertExistsAndLoadBean(t, &organization.Team{ID: 2})
	team.LowerName = "owners"
	team.Name = "Owners"
	team.Description = strings.Repeat("A long description!", 100)
	err := UpdateTeam(db.DefaultContext, team, true, false)
	assert.True(t, organization.IsErrTeamAlreadyExist(err))

	unittest.CheckConsistencyFor(t, &organization.Team{ID: team.ID})
}

func TestDeleteTeam(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	team := unittest.AssertExistsAndLoadBean(t, &organization.Team{ID: 2})
	require.NoError(t, DeleteTeam(db.DefaultContext, team))
	unittest.AssertNotExistsBean(t, &organization.Team{ID: team.ID})
	unittest.AssertNotExistsBean(t, &organization.TeamRepo{TeamID: team.ID})
	unittest.AssertNotExistsBean(t, &organization.TeamUser{TeamID: team.ID})

	// check that team members don't have "leftover" access to repos
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 4})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 3})
	accessMode, err := access_model.AccessLevel(db.DefaultContext, user, repo)
	require.NoError(t, err)
	assert.Less(t, accessMode, perm.AccessModeWrite)
}

func TestAddTeamMember(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	test := func(teamID, userID int64) {
		team := unittest.AssertExistsAndLoadBean(t, &organization.Team{ID: teamID})
		require.NoError(t, AddTeamMember(db.DefaultContext, team, userID))
		unittest.AssertExistsAndLoadBean(t, &organization.TeamUser{UID: userID, TeamID: teamID})
		unittest.CheckConsistencyFor(t, &organization.Team{ID: teamID}, &user_model.User{ID: team.OrgID})
	}
	test(1, 2)
	test(1, 4)
	test(3, 2)
}

func TestRemoveTeamMember(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	testSuccess := func(teamID, userID int64) {
		team := unittest.AssertExistsAndLoadBean(t, &organization.Team{ID: teamID})
		require.NoError(t, RemoveTeamMember(db.DefaultContext, team, userID))
		unittest.AssertNotExistsBean(t, &organization.TeamUser{UID: userID, TeamID: teamID})
		unittest.CheckConsistencyFor(t, &organization.Team{ID: teamID})
	}
	testSuccess(1, 4)
	testSuccess(2, 2)
	testSuccess(3, 2)
	testSuccess(3, unittest.NonexistentID)

	team := unittest.AssertExistsAndLoadBean(t, &organization.Team{ID: 1})
	err := RemoveTeamMember(db.DefaultContext, team, 2)
	assert.True(t, organization.IsErrLastOrgOwner(err))
}

func TestRepository_RecalculateAccesses3(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	team5 := unittest.AssertExistsAndLoadBean(t, &organization.Team{ID: 5})
	user29 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 29})

	has, err := db.GetEngine(db.DefaultContext).Get(&access_model.Access{UserID: 29, RepoID: 23})
	require.NoError(t, err)
	assert.False(t, has)

	// adding user29 to team5 should add an explicit access row for repo 23
	// even though repo 23 is public
	require.NoError(t, AddTeamMember(db.DefaultContext, team5, user29.ID))

	has, err = db.GetEngine(db.DefaultContext).Get(&access_model.Access{UserID: 29, RepoID: 23})
	require.NoError(t, err)
	assert.True(t, has)
}
