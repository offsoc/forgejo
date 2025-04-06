// Copyright 2024 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package v1_22 //nolint

import (
	"slices"
	"testing"

	migration_tests "forgejo.org/models/migrations/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"xorm.io/xorm/schemas"
)

func Test_AddUniqueIndexForProjectIssue(t *testing.T) {
	type ProjectIssue struct { //revive:disable-line:exported
		ID        int64 `xorm:"pk autoincr"`
		IssueID   int64 `xorm:"INDEX"`
		ProjectID int64 `xorm:"INDEX"`
	}

	// Prepare and load the testing database
	x, deferable := migration_tests.PrepareTestEnv(t, 0, new(ProjectIssue))
	defer deferable()
	if x == nil || t.Failed() {
		return
	}

	cnt, err := x.Table("project_issue").Where("project_id=1 AND issue_id=1").Count()
	require.NoError(t, err)
	assert.EqualValues(t, 2, cnt)

	require.NoError(t, AddUniqueIndexForProjectIssue(x))

	cnt, err = x.Table("project_issue").Where("project_id=1 AND issue_id=1").Count()
	require.NoError(t, err)
	assert.EqualValues(t, 1, cnt)

	tables, err := x.DBMetas()
	require.NoError(t, err)
	assert.Len(t, tables, 1)
	found := false
	for _, index := range tables[0].Indexes {
		if index.Type == schemas.UniqueType {
			found = true
			slices.Equal(index.Cols, []string{"project_id", "issue_id"})
			break
		}
	}
	assert.True(t, found)
}
