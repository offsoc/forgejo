// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user_test

import (
	"testing"

	"forgejo.org/models/db"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsFollowing(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	assert.True(t, user_model.IsFollowing(db.DefaultContext, 4, 2))
	assert.False(t, user_model.IsFollowing(db.DefaultContext, 2, 4))
	assert.False(t, user_model.IsFollowing(db.DefaultContext, 5, unittest.NonexistentID))
	assert.False(t, user_model.IsFollowing(db.DefaultContext, unittest.NonexistentID, 5))
	assert.False(t, user_model.IsFollowing(db.DefaultContext, unittest.NonexistentID, unittest.NonexistentID))
}
