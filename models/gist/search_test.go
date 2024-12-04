// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package gist_test

import (
	"testing"

	"code.gitea.io/gitea/models/db"
	gist_model "code.gitea.io/gitea/models/gist"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearchGist(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	admin := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})

	t.Run("AllGuest", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		gists, count, err := gist_model.SearchGist(db.DefaultContext, &gist_model.SearchGistOptions{})
		require.NoError(t, err)

		assert.Len(t, gists, 2)
		assert.Equal(t, int64(2), count)
		assert.Equal(t, int64(1), gists[0].ID)
		assert.Equal(t, int64(4), gists[1].ID)
	})

	t.Run("AllUser", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		gists, count, err := gist_model.SearchGist(db.DefaultContext, &gist_model.SearchGistOptions{Actor: user})
		require.NoError(t, err)

		assert.Len(t, gists, 4)
		assert.Equal(t, int64(4), count)
		assert.Equal(t, int64(1), gists[0].ID)
		assert.Equal(t, int64(2), gists[1].ID)
		assert.Equal(t, int64(3), gists[2].ID)
		assert.Equal(t, int64(4), gists[3].ID)
	})

	t.Run("AllAdmin", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		gists, count, err := gist_model.SearchGist(db.DefaultContext, &gist_model.SearchGistOptions{Actor: admin})
		require.NoError(t, err)

		assert.Len(t, gists, 4)
		assert.Equal(t, int64(4), count)
		assert.Equal(t, int64(1), gists[0].ID)
		assert.Equal(t, int64(2), gists[1].ID)
		assert.Equal(t, int64(3), gists[2].ID)
		assert.Equal(t, int64(4), gists[3].ID)
	})

	t.Run("OwnerID", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		gists, count, err := gist_model.SearchGist(db.DefaultContext, &gist_model.SearchGistOptions{OwnerID: 2})
		require.NoError(t, err)

		assert.Len(t, gists, 1)
		assert.Equal(t, int64(1), count)
		assert.Equal(t, int64(1), gists[0].ID)
	})

	t.Run("Keyword", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		gists, count, err := gist_model.SearchGist(db.DefaultContext, &gist_model.SearchGistOptions{Keyword: "another"})
		require.NoError(t, err)

		assert.Len(t, gists, 1)
		assert.Equal(t, int64(1), count)
		assert.Equal(t, int64(4), gists[0].ID)
	})
}

func TestCountGist(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	count, err := gist_model.CountGist(db.DefaultContext, &gist_model.SearchGistOptions{})
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
}
