// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repository

import (
	"testing"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/organization"
	perm_model "code.gitea.io/gitea/models/perm"
	access_model "code.gitea.io/gitea/models/perm/access"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/unit"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepository_AddCollaborator(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	testSuccess := func(repoID, userID int64) {
		repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: repoID})
		require.NoError(t, repo.LoadOwner(db.DefaultContext))
		user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: userID})
		require.NoError(t, AddCollaborator(db.DefaultContext, repo, user))
		unittest.CheckConsistencyFor(t, &repo_model.Repository{ID: repoID}, &user_model.User{ID: userID})
	}
	testSuccess(1, 4)
	testSuccess(1, 4)
	testSuccess(3, 4)
}

func TestRepository_AddCollaborator_IsBlocked(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	testSuccess := func(repoID, userID int64) {
		repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: repoID})
		require.NoError(t, repo.LoadOwner(db.DefaultContext))
		user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: userID})

		// Owner blocked user.
		unittest.AssertSuccessfulInsert(t, &user_model.BlockedUser{UserID: repo.OwnerID, BlockID: userID})
		require.ErrorIs(t, AddCollaborator(db.DefaultContext, repo, user), user_model.ErrBlockedByUser)
		unittest.CheckConsistencyFor(t, &repo_model.Repository{ID: repoID}, &user_model.User{ID: userID})
		_, err := db.DeleteByBean(db.DefaultContext, &user_model.BlockedUser{UserID: repo.OwnerID, BlockID: userID})
		require.NoError(t, err)

		// User has owner blocked.
		unittest.AssertSuccessfulInsert(t, &user_model.BlockedUser{UserID: userID, BlockID: repo.OwnerID})
		require.ErrorIs(t, AddCollaborator(db.DefaultContext, repo, user), user_model.ErrBlockedByUser)
		unittest.CheckConsistencyFor(t, &repo_model.Repository{ID: repoID}, &user_model.User{ID: userID})
	}
	// Ensure idempotency (public repository).
	testSuccess(1, 4)
	testSuccess(1, 4)
	// Add collaborator to private repository.
	testSuccess(3, 4)
}

func TestRepoPermissionPublicNonOrgRepo(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// public non-organization repo
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 4})
	require.NoError(t, repo.LoadUnits(db.DefaultContext))

	// plain user
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	perm, err := access_model.GetUserRepoPermission(db.DefaultContext, repo, user)
	require.NoError(t, err)
	for _, unit := range repo.Units {
		assert.True(t, perm.CanRead(unit.Type))
		assert.False(t, perm.CanWrite(unit.Type))
	}

	// change to collaborator
	require.NoError(t, AddCollaborator(db.DefaultContext, repo, user))
	perm, err = access_model.GetUserRepoPermission(db.DefaultContext, repo, user)
	require.NoError(t, err)
	for _, unit := range repo.Units {
		assert.True(t, perm.CanRead(unit.Type))
		assert.True(t, perm.CanWrite(unit.Type))
	}

	// collaborator
	collaborator := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 4})
	perm, err = access_model.GetUserRepoPermission(db.DefaultContext, repo, collaborator)
	require.NoError(t, err)
	for _, unit := range repo.Units {
		assert.True(t, perm.CanRead(unit.Type))
		assert.True(t, perm.CanWrite(unit.Type))
	}

	// owner
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 5})
	perm, err = access_model.GetUserRepoPermission(db.DefaultContext, repo, owner)
	require.NoError(t, err)
	for _, unit := range repo.Units {
		assert.True(t, perm.CanRead(unit.Type))
		assert.True(t, perm.CanWrite(unit.Type))
	}

	// admin
	admin := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
	perm, err = access_model.GetUserRepoPermission(db.DefaultContext, repo, admin)
	require.NoError(t, err)
	for _, unit := range repo.Units {
		assert.True(t, perm.CanRead(unit.Type))
		assert.True(t, perm.CanWrite(unit.Type))
	}
}

func TestRepoPermissionPrivateNonOrgRepo(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// private non-organization repo
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 2})
	require.NoError(t, repo.LoadUnits(db.DefaultContext))

	// plain user
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 4})
	perm, err := access_model.GetUserRepoPermission(db.DefaultContext, repo, user)
	require.NoError(t, err)
	for _, unit := range repo.Units {
		assert.False(t, perm.CanRead(unit.Type))
		assert.False(t, perm.CanWrite(unit.Type))
	}

	// change to collaborator to default write access
	require.NoError(t, AddCollaborator(db.DefaultContext, repo, user))
	perm, err = access_model.GetUserRepoPermission(db.DefaultContext, repo, user)
	require.NoError(t, err)
	for _, unit := range repo.Units {
		assert.True(t, perm.CanRead(unit.Type))
		assert.True(t, perm.CanWrite(unit.Type))
	}

	require.NoError(t, repo_model.ChangeCollaborationAccessMode(db.DefaultContext, repo, user.ID, perm_model.AccessModeRead))
	perm, err = access_model.GetUserRepoPermission(db.DefaultContext, repo, user)
	require.NoError(t, err)
	for _, unit := range repo.Units {
		assert.True(t, perm.CanRead(unit.Type))
		assert.False(t, perm.CanWrite(unit.Type))
	}

	// owner
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	perm, err = access_model.GetUserRepoPermission(db.DefaultContext, repo, owner)
	require.NoError(t, err)
	for _, unit := range repo.Units {
		assert.True(t, perm.CanRead(unit.Type))
		assert.True(t, perm.CanWrite(unit.Type))
	}

	// admin
	admin := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
	perm, err = access_model.GetUserRepoPermission(db.DefaultContext, repo, admin)
	require.NoError(t, err)
	for _, unit := range repo.Units {
		assert.True(t, perm.CanRead(unit.Type))
		assert.True(t, perm.CanWrite(unit.Type))
	}
}

func TestRepoPermissionPublicOrgRepo(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// public organization repo
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 32})
	require.NoError(t, repo.LoadUnits(db.DefaultContext))

	// plain user
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 5})
	perm, err := access_model.GetUserRepoPermission(db.DefaultContext, repo, user)
	require.NoError(t, err)
	for _, unit := range repo.Units {
		assert.True(t, perm.CanRead(unit.Type))
		assert.False(t, perm.CanWrite(unit.Type))
	}

	// change to collaborator to default write access
	require.NoError(t, AddCollaborator(db.DefaultContext, repo, user))
	perm, err = access_model.GetUserRepoPermission(db.DefaultContext, repo, user)
	require.NoError(t, err)
	for _, unit := range repo.Units {
		assert.True(t, perm.CanRead(unit.Type))
		assert.True(t, perm.CanWrite(unit.Type))
	}

	require.NoError(t, repo_model.ChangeCollaborationAccessMode(db.DefaultContext, repo, user.ID, perm_model.AccessModeRead))
	perm, err = access_model.GetUserRepoPermission(db.DefaultContext, repo, user)
	require.NoError(t, err)
	for _, unit := range repo.Units {
		assert.True(t, perm.CanRead(unit.Type))
		assert.False(t, perm.CanWrite(unit.Type))
	}

	// org member team owner
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	perm, err = access_model.GetUserRepoPermission(db.DefaultContext, repo, owner)
	require.NoError(t, err)
	for _, unit := range repo.Units {
		assert.True(t, perm.CanRead(unit.Type))
		assert.True(t, perm.CanWrite(unit.Type))
	}

	// org member team tester
	member := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 15})
	perm, err = access_model.GetUserRepoPermission(db.DefaultContext, repo, member)
	require.NoError(t, err)
	for _, unit := range repo.Units {
		assert.True(t, perm.CanRead(unit.Type))
	}
	assert.True(t, perm.CanWrite(unit.TypeIssues))
	assert.False(t, perm.CanWrite(unit.TypeCode))

	// admin
	admin := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
	perm, err = access_model.GetUserRepoPermission(db.DefaultContext, repo, admin)
	require.NoError(t, err)
	for _, unit := range repo.Units {
		assert.True(t, perm.CanRead(unit.Type))
		assert.True(t, perm.CanWrite(unit.Type))
	}
}

func TestRepoPermissionPrivateOrgRepo(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// private organization repo
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 24})
	require.NoError(t, repo.LoadUnits(db.DefaultContext))

	// plain user
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 5})
	perm, err := access_model.GetUserRepoPermission(db.DefaultContext, repo, user)
	require.NoError(t, err)
	for _, unit := range repo.Units {
		assert.False(t, perm.CanRead(unit.Type))
		assert.False(t, perm.CanWrite(unit.Type))
	}

	// change to collaborator to default write access
	require.NoError(t, AddCollaborator(db.DefaultContext, repo, user))
	perm, err = access_model.GetUserRepoPermission(db.DefaultContext, repo, user)
	require.NoError(t, err)
	for _, unit := range repo.Units {
		assert.True(t, perm.CanRead(unit.Type))
		assert.True(t, perm.CanWrite(unit.Type))
	}

	require.NoError(t, repo_model.ChangeCollaborationAccessMode(db.DefaultContext, repo, user.ID, perm_model.AccessModeRead))
	perm, err = access_model.GetUserRepoPermission(db.DefaultContext, repo, user)
	require.NoError(t, err)
	for _, unit := range repo.Units {
		assert.True(t, perm.CanRead(unit.Type))
		assert.False(t, perm.CanWrite(unit.Type))
	}

	// org member team owner
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 15})
	perm, err = access_model.GetUserRepoPermission(db.DefaultContext, repo, owner)
	require.NoError(t, err)
	for _, unit := range repo.Units {
		assert.True(t, perm.CanRead(unit.Type))
		assert.True(t, perm.CanWrite(unit.Type))
	}

	// update team information and then check permission
	team := unittest.AssertExistsAndLoadBean(t, &organization.Team{ID: 5})
	unittest.AssertSuccessfulDelete(t, &organization.TeamUnit{TeamID: team.ID})
	perm, err = access_model.GetUserRepoPermission(db.DefaultContext, repo, owner)
	require.NoError(t, err)
	for _, unit := range repo.Units {
		assert.True(t, perm.CanRead(unit.Type))
		assert.True(t, perm.CanWrite(unit.Type))
	}

	// org member team tester
	tester := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	perm, err = access_model.GetUserRepoPermission(db.DefaultContext, repo, tester)
	require.NoError(t, err)
	assert.True(t, perm.CanWrite(unit.TypeIssues))
	assert.False(t, perm.CanWrite(unit.TypeCode))
	assert.False(t, perm.CanRead(unit.TypeCode))

	// org member team reviewer
	reviewer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 20})
	perm, err = access_model.GetUserRepoPermission(db.DefaultContext, repo, reviewer)
	require.NoError(t, err)
	assert.False(t, perm.CanRead(unit.TypeIssues))
	assert.False(t, perm.CanWrite(unit.TypeCode))
	assert.True(t, perm.CanRead(unit.TypeCode))

	// admin
	admin := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
	perm, err = access_model.GetUserRepoPermission(db.DefaultContext, repo, admin)
	require.NoError(t, err)
	for _, unit := range repo.Units {
		assert.True(t, perm.CanRead(unit.Type))
		assert.True(t, perm.CanWrite(unit.Type))
	}
}
