// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package models

import (
	"testing"

	"forgejo.org/models/db"
	issues_model "forgejo.org/models/issues"
	"forgejo.org/models/unittest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckRepoStats(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	require.NoError(t, CheckRepoStats(db.DefaultContext))
}

func TestDoctorUserStarNum(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	require.NoError(t, DoctorUserStarNum(db.DefaultContext))
}

func Test_repoStatsCorrectIssueNumComments(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	issue2 := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 2})
	assert.NotNil(t, issue2)
	assert.Equal(t, 0, issue2.NumComments) // the fixture data is wrong, but we don't fix it here

	require.NoError(t, repoStatsCorrectIssueNumComments(db.DefaultContext, 2))
	// reload the issue
	issue2 = unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 2})
	assert.Equal(t, 1, issue2.NumComments)
}
