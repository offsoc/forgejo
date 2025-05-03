// Copyright 2024 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repository

import (
	"testing"

	"forgejo.org/models/db"
	git_model "forgejo.org/models/git"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSyncRepoBranches(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	_, err := db.GetEngine(db.DefaultContext).ID(1).Update(&repo_model.Repository{ObjectFormatName: "bad-fmt"})
	require.NoError(t, db.TruncateBeans(db.DefaultContext, &git_model.Branch{}))
	require.NoError(t, err)
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	assert.Equal(t, "bad-fmt", repo.ObjectFormatName)
	_, err = SyncRepoBranches(db.DefaultContext, 1, 0)
	require.NoError(t, err)
	repo = unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	assert.Equal(t, "sha1", repo.ObjectFormatName)
	branch, err := git_model.GetBranch(db.DefaultContext, 1, "master")
	require.NoError(t, err)
	assert.Equal(t, "master", branch.Name)
}
