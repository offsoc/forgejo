// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repository

import (
	"sync"
	"testing"

	"forgejo.org/models"
	activities_model "forgejo.org/models/activities"
	"forgejo.org/models/db"
	"forgejo.org/models/organization"
	access_model "forgejo.org/models/perm/access"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/util"
	"forgejo.org/services/feed"
	notify_service "forgejo.org/services/notify"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var notifySync sync.Once

func registerNotifier() {
	notifySync.Do(func() {
		notify_service.RegisterNotifier(feed.NewNotifier())
	})
}

func TestTransferOwnership(t *testing.T) {
	registerNotifier()

	require.NoError(t, unittest.PrepareTestDatabase())

	doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 3})
	repo.Owner = unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})
	require.NoError(t, TransferOwnership(db.DefaultContext, doer, doer, repo, nil))

	transferredRepo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 3})
	assert.EqualValues(t, 2, transferredRepo.OwnerID)

	exist, err := util.IsExist(repo_model.RepoPath("org3", "repo3"))
	require.NoError(t, err)
	assert.False(t, exist)
	exist, err = util.IsExist(repo_model.RepoPath("user2", "repo3"))
	require.NoError(t, err)
	assert.True(t, exist)
	unittest.AssertExistsAndLoadBean(t, &activities_model.Action{
		OpType:    activities_model.ActionTransferRepo,
		ActUserID: 2,
		RepoID:    3,
		Content:   "org3/repo3",
	})

	unittest.CheckConsistencyFor(t, &repo_model.Repository{}, &user_model.User{}, &organization.Team{})
}

func TestStartRepositoryTransferSetPermission(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 3})
	recipient := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 5})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 5})
	repo.Owner = unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})

	hasAccess, err := access_model.HasAccess(db.DefaultContext, recipient.ID, repo)
	require.NoError(t, err)
	assert.False(t, hasAccess)

	require.NoError(t, StartRepositoryTransfer(db.DefaultContext, doer, recipient, repo, nil))

	hasAccess, err = access_model.HasAccess(db.DefaultContext, recipient.ID, repo)
	require.NoError(t, err)
	assert.True(t, hasAccess)

	unittest.CheckConsistencyFor(t, &repo_model.Repository{}, &user_model.User{}, &organization.Team{})
}

func TestRepositoryTransfer(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 3})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 3})

	transfer, err := models.GetPendingRepositoryTransfer(db.DefaultContext, repo)
	require.NoError(t, err)
	assert.NotNil(t, transfer)

	// Cancel transfer
	require.NoError(t, CancelRepositoryTransfer(db.DefaultContext, repo))

	transfer, err = models.GetPendingRepositoryTransfer(db.DefaultContext, repo)
	require.Error(t, err)
	assert.Nil(t, transfer)
	assert.True(t, models.IsErrNoPendingTransfer(err))

	user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	require.NoError(t, models.CreatePendingRepositoryTransfer(db.DefaultContext, doer, user2, repo.ID, nil))

	transfer, err = models.GetPendingRepositoryTransfer(db.DefaultContext, repo)
	require.NoError(t, err)
	require.NoError(t, transfer.LoadAttributes(db.DefaultContext))
	assert.Equal(t, "user2", transfer.Recipient.Name)

	org6 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	// Only transfer can be started at any given time
	err = models.CreatePendingRepositoryTransfer(db.DefaultContext, doer, org6, repo.ID, nil)
	require.Error(t, err)
	assert.True(t, models.IsErrRepoTransferInProgress(err))

	// Unknown user
	err = models.CreatePendingRepositoryTransfer(db.DefaultContext, doer, &user_model.User{ID: 1000, LowerName: "user1000"}, repo.ID, nil)
	require.Error(t, err)

	// Cancel transfer
	require.NoError(t, CancelRepositoryTransfer(db.DefaultContext, repo))
}
