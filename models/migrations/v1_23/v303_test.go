// Copyright 2025 The Forgejo Authors.
// SPDX-License-Identifier: GPL-3.0-or-later

package v1_23 //nolint

import (
	"testing"

	migration_tests "forgejo.org/models/migrations/test"

	"github.com/stretchr/testify/require"
	"xorm.io/xorm/schemas"
)

func Test_GiteaLastDrop(t *testing.T) {
	type Badge struct {
		ID   int64 `xorm:"pk autoincr"`
		Slug string
	}

	x, deferable := migration_tests.PrepareTestEnv(t, 0, new(Badge))
	defer deferable()
	if x == nil || t.Failed() {
		return
	}

	getColumn := func() *schemas.Column {
		tables, err := x.DBMetas()
		require.NoError(t, err)
		require.Len(t, tables, 1)
		table := tables[0]
		require.Equal(t, "badge", table.Name)
		return table.GetColumn("slug")
	}

	require.NotNil(t, getColumn(), "slug column exists")
	require.NoError(t, GiteaLastDrop(x))
	require.Nil(t, getColumn(), "slug column was deleted")
	// idempotent
	require.NoError(t, GiteaLastDrop(x))
}
