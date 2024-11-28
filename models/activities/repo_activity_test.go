// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package activities

import (
	"testing"
	"time"

	"code.gitea.io/gitea/models/db"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/unittest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetActivityStats(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})

	stats, err := GetActivityStats(db.DefaultContext, repo, time.Unix(0, 0), true, true, true, true)
	require.NoError(t, err)

	assert.EqualValues(t, 2, stats.ActiveIssueCount())
	assert.EqualValues(t, 2, stats.OpenedIssueCount())
	assert.EqualValues(t, 0, stats.ClosedIssueCount())
	assert.EqualValues(t, 3, stats.ActivePRCount())
}
