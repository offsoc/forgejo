// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package v1_18 //nolint

import (
	"testing"

	"forgejo.org/models/issues"
	migration_tests "forgejo.org/models/migrations/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_UpdateOpenMilestoneCounts(t *testing.T) {
	type ExpectedMilestone issues.Milestone

	// Prepare and load the testing database
	x, deferable := migration_tests.PrepareTestEnv(t, 0, new(issues.Milestone), new(ExpectedMilestone), new(issues.Issue))
	defer deferable()
	if x == nil || t.Failed() {
		return
	}

	if err := UpdateOpenMilestoneCounts(x); err != nil {
		require.NoError(t, err)
		return
	}

	expected := []ExpectedMilestone{}
	err := x.Table("expected_milestone").Asc("id").Find(&expected)
	require.NoError(t, err)

	got := []issues.Milestone{}
	err = x.Table("milestone").Asc("id").Find(&got)
	require.NoError(t, err)

	for i, e := range expected {
		got := got[i]
		assert.Equal(t, e.ID, got.ID)
		assert.Equal(t, e.NumIssues, got.NumIssues)
		assert.Equal(t, e.NumClosedIssues, got.NumClosedIssues)
	}
}
