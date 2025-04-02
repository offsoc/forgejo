// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package v1_15 //nolint

import (
	"testing"

	migration_tests "forgejo.org/models/migrations/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_AddIssueResourceIndexTable(t *testing.T) {
	// Create the models used in the migration
	type Issue struct {
		ID     int64 `xorm:"pk autoincr"`
		RepoID int64 `xorm:"UNIQUE(s)"`
		Index  int64 `xorm:"UNIQUE(s)"`
	}

	// Prepare and load the testing database
	x, deferable := migration_tests.PrepareTestEnv(t, 0, new(Issue))
	if x == nil || t.Failed() {
		defer deferable()
		return
	}
	defer deferable()

	// Run the migration
	if err := AddIssueResourceIndexTable(x); err != nil {
		require.NoError(t, err)
		return
	}

	type ResourceIndex struct {
		GroupID  int64 `xorm:"pk"`
		MaxIndex int64 `xorm:"index"`
	}

	start := 0
	const batchSize = 1000
	for {
		indexes := make([]ResourceIndex, 0, batchSize)
		err := x.Table("issue_index").Limit(batchSize, start).Find(&indexes)
		require.NoError(t, err)

		for _, idx := range indexes {
			var maxIndex int
			has, err := x.SQL("SELECT max(`index`) FROM issue WHERE repo_id = ?", idx.GroupID).Get(&maxIndex)
			require.NoError(t, err)
			assert.True(t, has)
			assert.EqualValues(t, maxIndex, idx.MaxIndex)
		}
		if len(indexes) < batchSize {
			break
		}
		start += len(indexes)
	}
}
