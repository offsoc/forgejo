// Copyright 2024 The Forgejo Authors c/o Codeberg e.V.. All rights reserved.
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

func TestRepositoryFlags(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 10})

	// ********************
	// ** NEGATIVE TESTS **
	// ********************

	// Unless we add flags, the repo has none
	flags, err := repo.ListFlags(db.DefaultContext)
	require.NoError(t, err)
	assert.Empty(t, flags)

	// If the repo has no flags, it is not flagged
	flagged := repo.IsFlagged(db.DefaultContext)
	assert.False(t, flagged)

	// Trying to find a flag when there is none
	has := repo.HasFlag(db.DefaultContext, "foo")
	assert.False(t, has)

	// Trying to retrieve a non-existent flag indicates not found
	has, _, err = repo.GetFlag(db.DefaultContext, "foo")
	require.NoError(t, err)
	assert.False(t, has)

	// Deleting a non-existent flag fails
	deleted, err := repo.DeleteFlag(db.DefaultContext, "no-such-flag")
	require.NoError(t, err)
	assert.Equal(t, int64(0), deleted)

	// ********************
	// ** POSITIVE TESTS **
	// ********************

	// Adding a flag works
	err = repo.AddFlag(db.DefaultContext, "foo")
	require.NoError(t, err)

	// Adding it again fails
	err = repo.AddFlag(db.DefaultContext, "foo")
	require.Error(t, err)

	// Listing flags includes the one we added
	flags, err = repo.ListFlags(db.DefaultContext)
	require.NoError(t, err)
	assert.Len(t, flags, 1)
	assert.Equal(t, "foo", flags[0].Name)

	// With a flag added, the repo is flagged
	flagged = repo.IsFlagged(db.DefaultContext)
	assert.True(t, flagged)

	// The flag can be found
	has = repo.HasFlag(db.DefaultContext, "foo")
	assert.True(t, has)

	// Added flag can be retrieved
	_, flag, err := repo.GetFlag(db.DefaultContext, "foo")
	require.NoError(t, err)
	assert.Equal(t, "foo", flag.Name)

	// Deleting a flag works
	deleted, err = repo.DeleteFlag(db.DefaultContext, "foo")
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)

	// The list is now empty
	flags, err = repo.ListFlags(db.DefaultContext)
	require.NoError(t, err)
	assert.Empty(t, flags)

	// Replacing an empty list works
	err = repo.ReplaceAllFlags(db.DefaultContext, []string{"bar"})
	require.NoError(t, err)

	// The repo is now flagged with "bar"
	has = repo.HasFlag(db.DefaultContext, "bar")
	assert.True(t, has)

	// Replacing a tag set with another works
	err = repo.ReplaceAllFlags(db.DefaultContext, []string{"baz", "quux"})
	require.NoError(t, err)

	// The repo now has two tags
	flags, err = repo.ListFlags(db.DefaultContext)
	require.NoError(t, err)
	assert.Len(t, flags, 2)
	assert.Equal(t, "baz", flags[0].Name)
	assert.Equal(t, "quux", flags[1].Name)

	// Replacing flags with an empty set deletes all flags
	err = repo.ReplaceAllFlags(db.DefaultContext, []string{})
	require.NoError(t, err)

	// The repo is now unflagged
	flagged = repo.IsFlagged(db.DefaultContext)
	assert.False(t, flagged)
}
