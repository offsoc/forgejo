// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package convert

import (
	"testing"

	"forgejo.org/models/db"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	api "forgejo.org/modules/structs"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUser_ToUser(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	user1 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1, IsAdmin: true})

	apiUser := toUser(db.DefaultContext, user1, true, true)
	assert.True(t, apiUser.IsAdmin)
	assert.Contains(t, apiUser.AvatarURL, "://")

	user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2, IsAdmin: false})

	apiUser = toUser(db.DefaultContext, user2, true, true)
	assert.False(t, apiUser.IsAdmin)

	apiUser = toUser(db.DefaultContext, user1, false, false)
	assert.False(t, apiUser.IsAdmin)
	assert.Equal(t, api.VisibleTypePublic.String(), apiUser.Visibility)

	user31 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 31, IsAdmin: false, Visibility: api.VisibleTypePrivate})

	apiUser = toUser(db.DefaultContext, user31, true, true)
	assert.False(t, apiUser.IsAdmin)
	assert.Equal(t, api.VisibleTypePrivate.String(), apiUser.Visibility)
}
