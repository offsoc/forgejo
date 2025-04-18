// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"testing"

	"forgejo.org/models/auth"
	"forgejo.org/models/db"
	"forgejo.org/models/unittest"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_RegenerateSession(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	require.NoError(t, unittest.PrepareTestDatabase())

	key := "new_key890123456"  // it must be 16 characters long
	key2 := "new_key890123457" // it must be 16 characters
	exist, err := auth.ExistSession(db.DefaultContext, key)
	require.NoError(t, err)
	assert.False(t, exist)

	sess, err := auth.RegenerateSession(db.DefaultContext, "", key)
	require.NoError(t, err)
	assert.Equal(t, key, sess.Key)
	assert.Empty(t, sess.Data, 0)

	sess, err = auth.ReadSession(db.DefaultContext, key2)
	require.NoError(t, err)
	assert.Equal(t, key2, sess.Key)
	assert.Empty(t, sess.Data, 0)
}
