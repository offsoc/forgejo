// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package db_test

import (
	"testing"

	"forgejo.org/models/db"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"xorm.io/builder"
)

type mockListOptions struct {
	db.ListOptions
}

func (opts mockListOptions) IsListAll() bool {
	return true
}

func (opts mockListOptions) ToConds() builder.Cond {
	return builder.NewCond()
}

func TestFind(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	xe, err := unittest.GetXORMEngine()
	require.NoError(t, err)
	require.NoError(t, xe.Sync(&repo_model.RepoUnit{}))

	var repoUnitCount int
	_, err = db.GetEngine(db.DefaultContext).SQL("SELECT COUNT(*) FROM repo_unit").Get(&repoUnitCount)
	require.NoError(t, err)
	assert.NotEmpty(t, repoUnitCount)

	opts := mockListOptions{}
	repoUnits, err := db.Find[repo_model.RepoUnit](db.DefaultContext, opts)
	require.NoError(t, err)
	assert.Len(t, repoUnits, repoUnitCount)

	cnt, err := db.Count[repo_model.RepoUnit](db.DefaultContext, opts)
	require.NoError(t, err)
	assert.EqualValues(t, repoUnitCount, cnt)

	repoUnits, newCnt, err := db.FindAndCount[repo_model.RepoUnit](db.DefaultContext, opts)
	require.NoError(t, err)
	assert.Equal(t, cnt, newCnt)
	assert.Len(t, repoUnits, repoUnitCount)
}
