// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package git_test

import (
	"testing"

	"forgejo.org/models/db"
	git_model "forgejo.org/models/git"
	"forgejo.org/models/unittest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsUserAllowed(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	pt := &git_model.ProtectedTag{}
	allowed, err := git_model.IsUserAllowedModifyTag(db.DefaultContext, pt, 1)
	require.NoError(t, err)
	assert.False(t, allowed)

	pt = &git_model.ProtectedTag{
		AllowlistUserIDs: []int64{1},
	}
	allowed, err = git_model.IsUserAllowedModifyTag(db.DefaultContext, pt, 1)
	require.NoError(t, err)
	assert.True(t, allowed)

	allowed, err = git_model.IsUserAllowedModifyTag(db.DefaultContext, pt, 2)
	require.NoError(t, err)
	assert.False(t, allowed)

	pt = &git_model.ProtectedTag{
		AllowlistTeamIDs: []int64{1},
	}
	allowed, err = git_model.IsUserAllowedModifyTag(db.DefaultContext, pt, 1)
	require.NoError(t, err)
	assert.False(t, allowed)

	allowed, err = git_model.IsUserAllowedModifyTag(db.DefaultContext, pt, 2)
	require.NoError(t, err)
	assert.True(t, allowed)

	pt = &git_model.ProtectedTag{
		AllowlistUserIDs: []int64{1},
		AllowlistTeamIDs: []int64{1},
	}
	allowed, err = git_model.IsUserAllowedModifyTag(db.DefaultContext, pt, 1)
	require.NoError(t, err)
	assert.True(t, allowed)

	allowed, err = git_model.IsUserAllowedModifyTag(db.DefaultContext, pt, 2)
	require.NoError(t, err)
	assert.True(t, allowed)
}

func TestIsUserAllowedToControlTag(t *testing.T) {
	cases := []struct {
		name    string
		userid  int64
		allowed bool
	}{
		{
			name:    "test",
			userid:  1,
			allowed: true,
		},
		{
			name:    "test",
			userid:  3,
			allowed: true,
		},
		{
			name:    "gitea",
			userid:  1,
			allowed: true,
		},
		{
			name:    "gitea",
			userid:  3,
			allowed: false,
		},
		{
			name:    "test-gitea",
			userid:  1,
			allowed: true,
		},
		{
			name:    "test-gitea",
			userid:  3,
			allowed: false,
		},
		{
			name:    "gitea-test",
			userid:  1,
			allowed: true,
		},
		{
			name:    "gitea-test",
			userid:  3,
			allowed: true,
		},
		{
			name:    "v-1",
			userid:  1,
			allowed: false,
		},
		{
			name:    "v-1",
			userid:  2,
			allowed: true,
		},
		{
			name:    "release",
			userid:  1,
			allowed: false,
		},
	}

	t.Run("Glob", func(t *testing.T) {
		protectedTags := []*git_model.ProtectedTag{
			{
				NamePattern:      `*gitea`,
				AllowlistUserIDs: []int64{1},
			},
			{
				NamePattern:      `v-*`,
				AllowlistUserIDs: []int64{2},
			},
			{
				NamePattern: "release",
			},
		}

		for n, c := range cases {
			isAllowed, err := git_model.IsUserAllowedToControlTag(db.DefaultContext, protectedTags, c.name, c.userid)
			require.NoError(t, err)
			assert.Equal(t, c.allowed, isAllowed, "case %d: error should match", n)
		}
	})

	t.Run("Regex", func(t *testing.T) {
		protectedTags := []*git_model.ProtectedTag{
			{
				NamePattern:      `/gitea\z/`,
				AllowlistUserIDs: []int64{1},
			},
			{
				NamePattern:      `/\Av-/`,
				AllowlistUserIDs: []int64{2},
			},
			{
				NamePattern: "/release/",
			},
		}

		for n, c := range cases {
			isAllowed, err := git_model.IsUserAllowedToControlTag(db.DefaultContext, protectedTags, c.name, c.userid)
			require.NoError(t, err)
			assert.Equal(t, c.allowed, isAllowed, "case %d: error should match", n)
		}
	})
}
