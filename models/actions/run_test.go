// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package actions

import (
	"testing"
	"time"

	"forgejo.org/models/db"
	"forgejo.org/models/unittest"
	"forgejo.org/modules/timeutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetRunBefore(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// this repo is part of the test database requiring loading "repository.yml" in main_test.go
	var repoID int64 = 1

	workflowID := "test_workflow"

	// third completed run
	time1, err := time.Parse(time.RFC3339, "2024-07-31T15:47:55+08:00")
	require.NoError(t, err)
	timeutil.MockSet(time1)
	run1 := ActionRun{
		ID:         1,
		Index:      1,
		RepoID:     repoID,
		Stopped:    timeutil.TimeStampNow(),
		WorkflowID: workflowID,
	}

	// fourth completed run
	time2, err := time.Parse(time.RFC3339, "2024-08-31T15:47:55+08:00")
	require.NoError(t, err)
	timeutil.MockSet(time2)
	run2 := ActionRun{
		ID:         2,
		Index:      2,
		RepoID:     repoID,
		Stopped:    timeutil.TimeStampNow(),
		WorkflowID: workflowID,
	}

	// second completed run
	time3, err := time.Parse(time.RFC3339, "2024-07-31T15:47:54+08:00")
	require.NoError(t, err)
	timeutil.MockSet(time3)
	run3 := ActionRun{
		ID:         3,
		Index:      3,
		RepoID:     repoID,
		Stopped:    timeutil.TimeStampNow(),
		WorkflowID: workflowID,
	}

	// first completed run
	time4, err := time.Parse(time.RFC3339, "2024-06-30T15:47:54+08:00")
	require.NoError(t, err)
	timeutil.MockSet(time4)
	run4 := ActionRun{
		ID:         4,
		Index:      4,
		RepoID:     repoID,
		Stopped:    timeutil.TimeStampNow(),
		WorkflowID: workflowID,
	}
	require.NoError(t, db.Insert(db.DefaultContext, &run1))
	runBefore, err := GetRunBefore(db.DefaultContext, repoID, run1.Stopped)
	// there is no run before run1
	require.Error(t, err)
	require.Nil(t, runBefore)

	// now there is only run3 before run1
	require.NoError(t, db.Insert(db.DefaultContext, &run3))
	runBefore, err = GetRunBefore(db.DefaultContext, repoID, run1.Stopped)
	require.NoError(t, err)
	assert.Equal(t, run3.ID, runBefore.ID)

	// there still is only run3 before run1
	require.NoError(t, db.Insert(db.DefaultContext, &run2))
	runBefore, err = GetRunBefore(db.DefaultContext, repoID, run1.Stopped)
	require.NoError(t, err)
	assert.Equal(t, run3.ID, runBefore.ID)

	// run4 is further away from run1
	require.NoError(t, db.Insert(db.DefaultContext, &run4))
	runBefore, err = GetRunBefore(db.DefaultContext, repoID, run1.Stopped)
	require.NoError(t, err)
	assert.Equal(t, run3.ID, runBefore.ID)
}
