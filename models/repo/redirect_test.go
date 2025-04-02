// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo_test

import (
	"testing"

	"forgejo.org/models/db"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLookupRedirect(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	repoID, err := repo_model.LookupRedirect(db.DefaultContext, 2, "oldrepo1")
	require.NoError(t, err)
	assert.EqualValues(t, 1, repoID)

	_, err = repo_model.LookupRedirect(db.DefaultContext, unittest.NonexistentID, "doesnotexist")
	assert.True(t, repo_model.IsErrRedirectNotExist(err))
}

func TestNewRedirect(t *testing.T) {
	// redirect to a completely new name
	require.NoError(t, unittest.PrepareTestDatabase())

	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	require.NoError(t, repo_model.NewRedirect(db.DefaultContext, repo.OwnerID, repo.ID, repo.Name, "newreponame"))

	unittest.AssertExistsAndLoadBean(t, &repo_model.Redirect{
		OwnerID:        repo.OwnerID,
		LowerName:      repo.LowerName,
		RedirectRepoID: repo.ID,
	})
	unittest.AssertExistsAndLoadBean(t, &repo_model.Redirect{
		OwnerID:        repo.OwnerID,
		LowerName:      "oldrepo1",
		RedirectRepoID: repo.ID,
	})
}

func TestNewRedirect2(t *testing.T) {
	// redirect to previously used name
	require.NoError(t, unittest.PrepareTestDatabase())

	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	require.NoError(t, repo_model.NewRedirect(db.DefaultContext, repo.OwnerID, repo.ID, repo.Name, "oldrepo1"))

	unittest.AssertExistsAndLoadBean(t, &repo_model.Redirect{
		OwnerID:        repo.OwnerID,
		LowerName:      repo.LowerName,
		RedirectRepoID: repo.ID,
	})
	unittest.AssertNotExistsBean(t, &repo_model.Redirect{
		OwnerID:        repo.OwnerID,
		LowerName:      "oldrepo1",
		RedirectRepoID: repo.ID,
	})
}

func TestNewRedirect3(t *testing.T) {
	// redirect for a previously-unredirected repo
	require.NoError(t, unittest.PrepareTestDatabase())

	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 2})
	require.NoError(t, repo_model.NewRedirect(db.DefaultContext, repo.OwnerID, repo.ID, repo.Name, "newreponame"))

	unittest.AssertExistsAndLoadBean(t, &repo_model.Redirect{
		OwnerID:        repo.OwnerID,
		LowerName:      repo.LowerName,
		RedirectRepoID: repo.ID,
	})
}
