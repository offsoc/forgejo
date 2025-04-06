// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issues_test

import (
	"testing"

	"forgejo.org/models/db"
	issues_model "forgejo.org/models/issues"
	"forgejo.org/models/unittest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetIssueStats(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	ids, err := issues_model.GetIssueIDsByRepoID(db.DefaultContext, 1)
	require.NoError(t, err)

	stats, err := issues_model.GetIssueStats(db.DefaultContext, &issues_model.IssuesOptions{IssueIDs: ids})
	require.NoError(t, err)

	assert.Equal(t, int64(4), stats.OpenCount)
	assert.Equal(t, int64(1), stats.ClosedCount)
	assert.Equal(t, int64(5), stats.AllCount)
	assert.Equal(t, int64(0), stats.YourRepositoriesCount)
	assert.Equal(t, int64(0), stats.AssignCount)
	assert.Equal(t, int64(0), stats.CreateCount)
	assert.Equal(t, int64(0), stats.MentionCount)
	assert.Equal(t, int64(0), stats.ReviewRequestedCount)
	assert.Equal(t, int64(0), stats.ReviewedCount)
}
