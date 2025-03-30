// Copyright 2023 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package auth_test

import (
	"testing"
	"time"

	"forgejo.org/models/auth"
	"forgejo.org/models/db"
	"forgejo.org/models/unittest"
	"forgejo.org/modules/timeutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthSession(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	defer timeutil.MockUnset()

	key := "I-Like-Free-Software"

	t.Run("Create Session", func(t *testing.T) {
		// Ensure it doesn't exist.
		ok, err := auth.ExistSession(db.DefaultContext, key)
		require.NoError(t, err)
		assert.False(t, ok)

		preCount, err := auth.CountSessions(db.DefaultContext)
		require.NoError(t, err)

		now := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
		timeutil.MockSet(now)

		// New session is created.
		sess, err := auth.ReadSession(db.DefaultContext, key)
		require.NoError(t, err)
		assert.Equal(t, key, sess.Key)
		assert.Empty(t, sess.Data)
		assert.EqualValues(t, now.Unix(), sess.Expiry)

		// Ensure it exists.
		ok, err = auth.ExistSession(db.DefaultContext, key)
		require.NoError(t, err)
		assert.True(t, ok)

		// Ensure the session is taken into account for count..
		postCount, err := auth.CountSessions(db.DefaultContext)
		require.NoError(t, err)
		assert.Greater(t, postCount, preCount)
	})

	t.Run("Update session", func(t *testing.T) {
		data := []byte{0xba, 0xdd, 0xc0, 0xde}
		now := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)
		timeutil.MockSet(now)

		// Update session.
		err := auth.UpdateSession(db.DefaultContext, key, data)
		require.NoError(t, err)

		timeutil.MockSet(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC))

		// Read updated session.
		// Ensure data is updated and expiry is set from the update session call.
		sess, err := auth.ReadSession(db.DefaultContext, key)
		require.NoError(t, err)
		assert.Equal(t, key, sess.Key)
		assert.Equal(t, data, sess.Data)
		assert.EqualValues(t, now.Unix(), sess.Expiry)

		timeutil.MockSet(now)
	})

	t.Run("Delete session", func(t *testing.T) {
		// Ensure it't exist.
		ok, err := auth.ExistSession(db.DefaultContext, key)
		require.NoError(t, err)
		assert.True(t, ok)

		preCount, err := auth.CountSessions(db.DefaultContext)
		require.NoError(t, err)

		err = auth.DestroySession(db.DefaultContext, key)
		require.NoError(t, err)

		// Ensure it doesn't exists.
		ok, err = auth.ExistSession(db.DefaultContext, key)
		require.NoError(t, err)
		assert.False(t, ok)

		// Ensure the session is taken into account for count..
		postCount, err := auth.CountSessions(db.DefaultContext)
		require.NoError(t, err)
		assert.Less(t, postCount, preCount)
	})

	t.Run("Cleanup sessions", func(t *testing.T) {
		timeutil.MockSet(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC))

		_, err := auth.ReadSession(db.DefaultContext, "sess-1")
		require.NoError(t, err)

		// One minute later.
		timeutil.MockSet(time.Date(2023, 1, 1, 0, 1, 0, 0, time.UTC))
		_, err = auth.ReadSession(db.DefaultContext, "sess-2")
		require.NoError(t, err)

		// 5 minutes, shouldn't clean up anything.
		err = auth.CleanupSessions(db.DefaultContext, 5*60)
		require.NoError(t, err)

		ok, err := auth.ExistSession(db.DefaultContext, "sess-1")
		require.NoError(t, err)
		assert.True(t, ok)

		ok, err = auth.ExistSession(db.DefaultContext, "sess-2")
		require.NoError(t, err)
		assert.True(t, ok)

		// 1 minute, should clean up sess-1.
		err = auth.CleanupSessions(db.DefaultContext, 60)
		require.NoError(t, err)

		ok, err = auth.ExistSession(db.DefaultContext, "sess-1")
		require.NoError(t, err)
		assert.False(t, ok)

		ok, err = auth.ExistSession(db.DefaultContext, "sess-2")
		require.NoError(t, err)
		assert.True(t, ok)

		// Now, should clean up sess-2.
		err = auth.CleanupSessions(db.DefaultContext, 0)
		require.NoError(t, err)

		ok, err = auth.ExistSession(db.DefaultContext, "sess-2")
		require.NoError(t, err)
		assert.False(t, ok)
	})
}
