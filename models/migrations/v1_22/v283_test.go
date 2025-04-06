// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package v1_22 //nolint

import (
	"testing"

	migration_tests "forgejo.org/models/migrations/test"

	"github.com/stretchr/testify/require"
)

func Test_AddCombinedIndexToIssueUser(t *testing.T) {
	type IssueUser struct { // old struct
		ID          int64 `xorm:"pk autoincr"`
		UID         int64 `xorm:"INDEX"` // User ID.
		IssueID     int64 `xorm:"INDEX"`
		IsRead      bool
		IsMentioned bool
	}

	// Prepare and load the testing database
	x, deferable := migration_tests.PrepareTestEnv(t, 0, new(IssueUser))
	defer deferable()

	require.NoError(t, AddCombinedIndexToIssueUser(x))
}
