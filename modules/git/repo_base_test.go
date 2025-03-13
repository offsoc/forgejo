// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package git

import (
	"bufio"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This unit test relies on the implementation detail of CatFileBatch.
func TestCatFileBatch(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	repo, err := OpenRepository(ctx, "./tests/repos/repo1_bare")
	require.NoError(t, err)
	defer repo.Close()

	var wr WriteCloserError
	var r *bufio.Reader
	var cancel1 func()
	t.Run("Request cat file batch", func(t *testing.T) {
		assert.Nil(t, repo.batch)
		wr, r, cancel1, err = repo.CatFileBatch(ctx)
		require.NoError(t, err)
		assert.NotNil(t, repo.batch)
		assert.Equal(t, repo.batch.Writer, wr)
		assert.True(t, repo.batchInUse)
	})

	t.Run("Request temporary cat file batch", func(t *testing.T) {
		wr, r, cancel, err := repo.CatFileBatch(ctx)
		require.NoError(t, err)
		assert.NotEqual(t, repo.batch.Writer, wr)

		t.Run("Check temporary cat file batch", func(t *testing.T) {
			_, err = wr.Write([]byte("95bb4d39648ee7e325106df01a621c530863a653" + "\n"))
			require.NoError(t, err)

			sha, typ, size, err := ReadBatchLine(r)
			require.NoError(t, err)
			assert.Equal(t, "commit", typ)
			assert.EqualValues(t, []byte("95bb4d39648ee7e325106df01a621c530863a653"), sha)
			assert.EqualValues(t, 144, size)
		})

		cancel()
		assert.True(t, repo.batchInUse)
	})

	t.Run("Check cached cat file batch", func(t *testing.T) {
		_, err = wr.Write([]byte("95bb4d39648ee7e325106df01a621c530863a653" + "\n"))
		require.NoError(t, err)

		sha, typ, size, err := ReadBatchLine(r)
		require.NoError(t, err)
		assert.Equal(t, "commit", typ)
		assert.EqualValues(t, []byte("95bb4d39648ee7e325106df01a621c530863a653"), sha)
		assert.EqualValues(t, 144, size)
	})

	t.Run("Cancel cached cat file batch", func(t *testing.T) {
		cancel1()
		assert.False(t, repo.batchInUse)
		assert.NotNil(t, repo.batch)
	})

	t.Run("Request cached cat file batch", func(t *testing.T) {
		wr, _, _, err := repo.CatFileBatch(ctx)
		require.NoError(t, err)
		assert.NotNil(t, repo.batch)
		assert.Equal(t, repo.batch.Writer, wr)
		assert.True(t, repo.batchInUse)

		t.Run("Close git repo", func(t *testing.T) {
			require.NoError(t, repo.Close())
			assert.Nil(t, repo.batch)
		})

		_, err = wr.Write([]byte("95bb4d39648ee7e325106df01a621c530863a653" + "\n"))
		require.Error(t, err)
	})
}

// This unit test relies on the implementation detail of CatFileBatchCheck.
func TestCatFileBatchCheck(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	repo, err := OpenRepository(ctx, "./tests/repos/repo1_bare")
	require.NoError(t, err)
	defer repo.Close()

	var wr WriteCloserError
	var r *bufio.Reader
	var cancel1 func()
	t.Run("Request cat file batch check", func(t *testing.T) {
		assert.Nil(t, repo.check)
		wr, r, cancel1, err = repo.CatFileBatchCheck(ctx)
		require.NoError(t, err)
		assert.NotNil(t, repo.check)
		assert.Equal(t, repo.check.Writer, wr)
		assert.True(t, repo.checkInUse)
	})

	t.Run("Request temporary cat file batch check", func(t *testing.T) {
		wr, r, cancel, err := repo.CatFileBatchCheck(ctx)
		require.NoError(t, err)
		assert.NotEqual(t, repo.check.Writer, wr)

		t.Run("Check temporary cat file batch check", func(t *testing.T) {
			_, err = wr.Write([]byte("test" + "\n"))
			require.NoError(t, err)

			sha, typ, size, err := ReadBatchLine(r)
			require.NoError(t, err)
			assert.Equal(t, "tag", typ)
			assert.EqualValues(t, []byte("3ad28a9149a2864384548f3d17ed7f38014c9e8a"), sha)
			assert.EqualValues(t, 807, size)
		})

		cancel()
		assert.True(t, repo.checkInUse)
	})

	t.Run("Check cached cat file batch check", func(t *testing.T) {
		_, err = wr.Write([]byte("test" + "\n"))
		require.NoError(t, err)

		sha, typ, size, err := ReadBatchLine(r)
		require.NoError(t, err)
		assert.Equal(t, "tag", typ)
		assert.EqualValues(t, []byte("3ad28a9149a2864384548f3d17ed7f38014c9e8a"), sha)
		assert.EqualValues(t, 807, size)
	})

	t.Run("Cancel cached cat file batch check", func(t *testing.T) {
		cancel1()
		assert.False(t, repo.checkInUse)
		assert.NotNil(t, repo.check)
	})

	t.Run("Request cached cat file batch check", func(t *testing.T) {
		wr, _, _, err := repo.CatFileBatchCheck(ctx)
		require.NoError(t, err)
		assert.NotNil(t, repo.check)
		assert.Equal(t, repo.check.Writer, wr)
		assert.True(t, repo.checkInUse)

		t.Run("Close git repo", func(t *testing.T) {
			require.NoError(t, repo.Close())
			assert.Nil(t, repo.check)
		})

		_, err = wr.Write([]byte("test" + "\n"))
		require.Error(t, err)
	})
}
