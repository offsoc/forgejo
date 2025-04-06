// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package git

import (
	"context"
	"path/filepath"
	"testing"

	"forgejo.org/models/db"
	"forgejo.org/models/unittest"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIterateRepositoryIDsWithLFSMetaObjects(t *testing.T) {
	defer unittest.OverrideFixtures(
		unittest.FixturesOptions{
			Dir:  filepath.Join(setting.AppWorkPath, "models/fixtures/"),
			Base: setting.AppWorkPath,
			Dirs: []string{"models/git/TestIterateRepositoryIDsWithLFSMetaObjects/"},
		},
	)()
	require.NoError(t, unittest.PrepareTestDatabase())

	type repocount struct {
		repoid int64
		count  int64
	}
	expected := []repocount{{1, 1}, {54, 4}}

	t.Run("Normal batch size", func(t *testing.T) {
		defer test.MockVariableValue(&setting.Database.IterateBufferSize, 20)()
		cases := []repocount{}

		err := IterateRepositoryIDsWithLFSMetaObjects(db.DefaultContext, func(ctx context.Context, repoID, count int64) error {
			cases = append(cases, repocount{repoID, count})
			return nil
		})
		require.NoError(t, err)
		assert.Equal(t, expected, cases)
	})

	t.Run("Low batch size", func(t *testing.T) {
		defer test.MockVariableValue(&setting.Database.IterateBufferSize, 1)()
		cases := []repocount{}

		err := IterateRepositoryIDsWithLFSMetaObjects(db.DefaultContext, func(ctx context.Context, repoID, count int64) error {
			cases = append(cases, repocount{repoID, count})
			return nil
		})
		require.NoError(t, err)
		assert.Equal(t, expected, cases)
	})
}

func TestIterateLFSMetaObjectsForRepo(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	expectedIDs := []int64{1, 2, 3, 4}

	t.Run("Normal batch size", func(t *testing.T) {
		defer test.MockVariableValue(&setting.Database.IterateBufferSize, 20)()
		actualIDs := []int64{}

		err := IterateLFSMetaObjectsForRepo(db.DefaultContext, 54, func(ctx context.Context, lo *LFSMetaObject) error {
			actualIDs = append(actualIDs, lo.ID)
			return nil
		}, &IterateLFSMetaObjectsForRepoOptions{})
		require.NoError(t, err)
		assert.Equal(t, expectedIDs, actualIDs)
	})

	t.Run("Low batch size", func(t *testing.T) {
		defer test.MockVariableValue(&setting.Database.IterateBufferSize, 1)()
		actualIDs := []int64{}

		err := IterateLFSMetaObjectsForRepo(db.DefaultContext, 54, func(ctx context.Context, lo *LFSMetaObject) error {
			actualIDs = append(actualIDs, lo.ID)
			return nil
		}, &IterateLFSMetaObjectsForRepoOptions{})
		require.NoError(t, err)
		assert.Equal(t, expectedIDs, actualIDs)

		t.Run("Batch handles updates", func(t *testing.T) {
			actualIDs := []int64{}

			err := IterateLFSMetaObjectsForRepo(db.DefaultContext, 54, func(ctx context.Context, lo *LFSMetaObject) error {
				actualIDs = append(actualIDs, lo.ID)
				_, err := db.DeleteByID[LFSMetaObject](ctx, lo.ID)
				require.NoError(t, err)
				return nil
			}, &IterateLFSMetaObjectsForRepoOptions{})
			require.NoError(t, err)
			assert.Equal(t, expectedIDs, actualIDs)
		})
	})
}
