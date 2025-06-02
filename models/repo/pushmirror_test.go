// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo_test

import (
	"testing"
	"time"

	"forgejo.org/models/db"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	"forgejo.org/modules/timeutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPushMirrorsIterate(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	now := timeutil.TimeStampNow()

	db.Insert(db.DefaultContext, &repo_model.PushMirror{
		RemoteName:     "test-1",
		LastUpdateUnix: now,
		Interval:       1,
	})

	long, _ := time.ParseDuration("24h")
	db.Insert(db.DefaultContext, &repo_model.PushMirror{
		RemoteName:     "test-2",
		LastUpdateUnix: now,
		Interval:       long,
	})

	db.Insert(db.DefaultContext, &repo_model.PushMirror{
		RemoteName:     "test-3",
		LastUpdateUnix: now,
		Interval:       0,
	})

	repo_model.PushMirrorsIterate(db.DefaultContext, 1, func(idx int, bean any) error {
		m, ok := bean.(*repo_model.PushMirror)
		assert.True(t, ok)
		assert.Equal(t, "test-1", m.RemoteName)
		assert.Equal(t, m.RemoteName, m.GetRemoteName())
		return nil
	})
}

func TestPushMirrorPrivatekey(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	m := &repo_model.PushMirror{
		RemoteName: "test-privatekey",
	}
	require.NoError(t, db.Insert(db.DefaultContext, m))

	privateKey := []byte{0x00, 0x01, 0x02, 0x04, 0x08, 0x10}
	t.Run("Set privatekey", func(t *testing.T) {
		require.NoError(t, m.SetPrivatekey(db.DefaultContext, privateKey))
	})

	t.Run("Normal retrieval", func(t *testing.T) {
		actualPrivateKey, err := m.Privatekey()
		require.NoError(t, err)
		assert.Equal(t, privateKey, actualPrivateKey)
	})

	t.Run("Incorrect retrieval", func(t *testing.T) {
		m.ID++
		actualPrivateKey, err := m.Privatekey()
		require.Error(t, err)
		assert.Empty(t, actualPrivateKey)
	})
}
