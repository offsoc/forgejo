// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issues_test

import (
	"path/filepath"
	"testing"

	"forgejo.org/models/db"
	issues_model "forgejo.org/models/issues"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/timeutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCancelStopwatch(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	user1, err := user_model.GetUserByID(db.DefaultContext, 1)
	require.NoError(t, err)

	issue1, err := issues_model.GetIssueByID(db.DefaultContext, 1)
	require.NoError(t, err)
	issue2, err := issues_model.GetIssueByID(db.DefaultContext, 2)
	require.NoError(t, err)

	err = issues_model.CancelStopwatch(db.DefaultContext, user1, issue1)
	require.NoError(t, err)
	unittest.AssertNotExistsBean(t, &issues_model.Stopwatch{UserID: user1.ID, IssueID: issue1.ID})

	_ = unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{Type: issues_model.CommentTypeCancelTracking, PosterID: user1.ID, IssueID: issue1.ID})

	require.NoError(t, issues_model.CancelStopwatch(db.DefaultContext, user1, issue2))
}

func TestStopwatchExists(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	assert.True(t, issues_model.StopwatchExists(db.DefaultContext, 1, 1))
	assert.False(t, issues_model.StopwatchExists(db.DefaultContext, 1, 2))
}

func TestHasUserStopwatch(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	exists, sw, _, err := issues_model.HasUserStopwatch(db.DefaultContext, 1)
	require.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, int64(1), sw.ID)

	exists, _, _, err = issues_model.HasUserStopwatch(db.DefaultContext, 3)
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestCreateOrStopIssueStopwatch(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	user2, err := user_model.GetUserByID(db.DefaultContext, 2)
	require.NoError(t, err)
	org3, err := user_model.GetUserByID(db.DefaultContext, 3)
	require.NoError(t, err)

	issue1, err := issues_model.GetIssueByID(db.DefaultContext, 1)
	require.NoError(t, err)
	issue2, err := issues_model.GetIssueByID(db.DefaultContext, 2)
	require.NoError(t, err)

	require.NoError(t, issues_model.CreateOrStopIssueStopwatch(db.DefaultContext, org3, issue1))
	sw := unittest.AssertExistsAndLoadBean(t, &issues_model.Stopwatch{UserID: 3, IssueID: 1})
	assert.LessOrEqual(t, sw.CreatedUnix, timeutil.TimeStampNow())

	require.NoError(t, issues_model.CreateOrStopIssueStopwatch(db.DefaultContext, user2, issue2))
	unittest.AssertNotExistsBean(t, &issues_model.Stopwatch{UserID: 2, IssueID: 2})
	unittest.AssertExistsAndLoadBean(t, &issues_model.TrackedTime{UserID: 2, IssueID: 2})
}

func TestGetUIDsAndStopwatch(t *testing.T) {
	defer unittest.OverrideFixtures(
		unittest.FixturesOptions{
			Dir:  filepath.Join(setting.AppWorkPath, "models/fixtures/"),
			Base: setting.AppWorkPath,
			Dirs: []string{"models/issues/TestGetUIDsAndStopwatch/"},
		},
	)()
	require.NoError(t, unittest.PrepareTestDatabase())

	uidStopwatches, err := issues_model.GetUIDsAndStopwatch(db.DefaultContext)
	require.NoError(t, err)
	assert.Equal(t, map[int64][]*issues_model.Stopwatch{
		1: {
			{
				ID:          1,
				UserID:      1,
				IssueID:     1,
				CreatedUnix: timeutil.TimeStamp(1500988001),
			},
			{
				ID:          3,
				UserID:      1,
				IssueID:     2,
				CreatedUnix: timeutil.TimeStamp(1500988004),
			},
		},
		2: {
			{
				ID:          2,
				UserID:      2,
				IssueID:     2,
				CreatedUnix: timeutil.TimeStamp(1500988002),
			},
		},
	}, uidStopwatches)
}
