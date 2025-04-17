// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package db_test

import (
	"context"
	"testing"

	"forgejo.org/models/db"
	"forgejo.org/models/unittest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInTransaction(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	assert.False(t, db.InTransaction(db.DefaultContext))
	require.NoError(t, db.WithTx(db.DefaultContext, func(ctx context.Context) error {
		assert.True(t, db.InTransaction(ctx))
		return nil
	}))

	ctx, committer, err := db.TxContext(db.DefaultContext)
	require.NoError(t, err)
	defer committer.Close()
	assert.True(t, db.InTransaction(ctx))
	require.NoError(t, db.WithTx(ctx, func(ctx context.Context) error {
		assert.True(t, db.InTransaction(ctx))
		return nil
	}))
}

func TestTxContext(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	{ // create new transaction
		ctx, committer, err := db.TxContext(db.DefaultContext)
		require.NoError(t, err)
		assert.True(t, db.InTransaction(ctx))
		require.NoError(t, committer.Commit())
	}

	{ // reuse the transaction created by TxContext and commit it
		ctx, committer, err := db.TxContext(db.DefaultContext)
		engine := db.GetEngine(ctx)
		require.NoError(t, err)
		assert.True(t, db.InTransaction(ctx))
		{
			ctx, committer, err := db.TxContext(ctx)
			require.NoError(t, err)
			assert.True(t, db.InTransaction(ctx))
			assert.Equal(t, engine, db.GetEngine(ctx))
			require.NoError(t, committer.Commit())
		}
		require.NoError(t, committer.Commit())
	}

	{ // reuse the transaction created by TxContext and close it
		ctx, committer, err := db.TxContext(db.DefaultContext)
		engine := db.GetEngine(ctx)
		require.NoError(t, err)
		assert.True(t, db.InTransaction(ctx))
		{
			ctx, committer, err := db.TxContext(ctx)
			require.NoError(t, err)
			assert.True(t, db.InTransaction(ctx))
			assert.Equal(t, engine, db.GetEngine(ctx))
			require.NoError(t, committer.Close())
		}
		require.NoError(t, committer.Close())
	}

	{ // reuse the transaction created by WithTx
		require.NoError(t, db.WithTx(db.DefaultContext, func(ctx context.Context) error {
			assert.True(t, db.InTransaction(ctx))
			{
				ctx, committer, err := db.TxContext(ctx)
				require.NoError(t, err)
				assert.True(t, db.InTransaction(ctx))
				require.NoError(t, committer.Commit())
			}
			return nil
		}))
	}
}
