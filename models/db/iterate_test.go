// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package db_test

import (
	"context"
	"testing"

	"forgejo.org/models/db"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIterate(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	xe, err := unittest.GetXORMEngine()
	require.NoError(t, err)
	require.NoError(t, xe.Sync(&repo_model.RepoUnit{}))

	cnt, err := db.GetEngine(db.DefaultContext).Count(&repo_model.RepoUnit{})
	require.NoError(t, err)

	var repoUnitCnt int
	err = db.Iterate(db.DefaultContext, nil, func(ctx context.Context, repo *repo_model.RepoUnit) error {
		repoUnitCnt++
		return nil
	})
	require.NoError(t, err)
	assert.EqualValues(t, cnt, repoUnitCnt)

	err = db.Iterate(db.DefaultContext, nil, func(ctx context.Context, repoUnit *repo_model.RepoUnit) error {
		has, err := db.ExistByID[repo_model.RepoUnit](ctx, repoUnit.ID)
		if err != nil {
			return err
		}
		if !has {
			return db.ErrNotExist{Resource: "repo_unit", ID: repoUnit.ID}
		}
		return nil
	})
	require.NoError(t, err)
}
