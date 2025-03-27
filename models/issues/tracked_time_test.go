// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issues_test

import (
	"testing"
	"time"

	"forgejo.org/models/db"
	issues_model "forgejo.org/models/issues"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/optional"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddTime(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	org3, err := user_model.GetUserByID(db.DefaultContext, 3)
	require.NoError(t, err)

	issue1, err := issues_model.GetIssueByID(db.DefaultContext, 1)
	require.NoError(t, err)

	// 3661 = 1h 1min 1s
	trackedTime, err := issues_model.AddTime(db.DefaultContext, org3, issue1, 3661, time.Now())
	require.NoError(t, err)
	assert.Equal(t, int64(3), trackedTime.UserID)
	assert.Equal(t, int64(1), trackedTime.IssueID)
	assert.Equal(t, int64(3661), trackedTime.Time)

	tt := unittest.AssertExistsAndLoadBean(t, &issues_model.TrackedTime{UserID: 3, IssueID: 1})
	assert.Equal(t, int64(3661), tt.Time)

	comment := unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{Type: issues_model.CommentTypeAddTimeManual, PosterID: 3, IssueID: 1})
	assert.Equal(t, "|3661", comment.Content)
}

func TestGetTrackedTimes(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// by Issue
	times, err := issues_model.GetTrackedTimes(db.DefaultContext, &issues_model.FindTrackedTimesOptions{IssueID: 1})
	require.NoError(t, err)
	assert.Len(t, times, 1)
	assert.Equal(t, int64(400), times[0].Time)

	times, err = issues_model.GetTrackedTimes(db.DefaultContext, &issues_model.FindTrackedTimesOptions{IssueID: -1})
	require.NoError(t, err)
	assert.Empty(t, times)

	// by User
	times, err = issues_model.GetTrackedTimes(db.DefaultContext, &issues_model.FindTrackedTimesOptions{UserID: 1})
	require.NoError(t, err)
	assert.Len(t, times, 3)
	assert.Equal(t, int64(400), times[0].Time)

	times, err = issues_model.GetTrackedTimes(db.DefaultContext, &issues_model.FindTrackedTimesOptions{UserID: 3})
	require.NoError(t, err)
	assert.Empty(t, times)

	// by Repo
	times, err = issues_model.GetTrackedTimes(db.DefaultContext, &issues_model.FindTrackedTimesOptions{RepositoryID: 2})
	require.NoError(t, err)
	assert.Len(t, times, 3)
	assert.Equal(t, int64(1), times[0].Time)
	issue, err := issues_model.GetIssueByID(db.DefaultContext, times[0].IssueID)
	require.NoError(t, err)
	assert.Equal(t, int64(2), issue.RepoID)

	times, err = issues_model.GetTrackedTimes(db.DefaultContext, &issues_model.FindTrackedTimesOptions{RepositoryID: 1})
	require.NoError(t, err)
	assert.Len(t, times, 5)

	times, err = issues_model.GetTrackedTimes(db.DefaultContext, &issues_model.FindTrackedTimesOptions{RepositoryID: 10})
	require.NoError(t, err)
	assert.Empty(t, times)
}

func TestTotalTimesForEachUser(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	total, err := issues_model.TotalTimesForEachUser(db.DefaultContext, &issues_model.FindTrackedTimesOptions{IssueID: 1})
	require.NoError(t, err)
	assert.Len(t, total, 1)
	for user, time := range total {
		assert.EqualValues(t, 1, user.ID)
		assert.EqualValues(t, 400, time)
	}

	total, err = issues_model.TotalTimesForEachUser(db.DefaultContext, &issues_model.FindTrackedTimesOptions{IssueID: 2})
	require.NoError(t, err)
	assert.Len(t, total, 2)
	for user, time := range total {
		if user.ID == 2 {
			assert.EqualValues(t, 3662, time)
		} else if user.ID == 1 {
			assert.EqualValues(t, 20, time)
		} else {
			require.Error(t, assert.AnError)
		}
	}

	total, err = issues_model.TotalTimesForEachUser(db.DefaultContext, &issues_model.FindTrackedTimesOptions{IssueID: 5})
	require.NoError(t, err)
	assert.Len(t, total, 1)
	for user, time := range total {
		assert.EqualValues(t, 2, user.ID)
		assert.EqualValues(t, 1, time)
	}

	total, err = issues_model.TotalTimesForEachUser(db.DefaultContext, &issues_model.FindTrackedTimesOptions{IssueID: 4})
	require.NoError(t, err)
	assert.Len(t, total, 2)
}

func TestGetIssueTotalTrackedTime(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	ttt, err := issues_model.GetIssueTotalTrackedTime(db.DefaultContext, &issues_model.IssuesOptions{MilestoneIDs: []int64{1}}, optional.Some(false))
	require.NoError(t, err)
	assert.EqualValues(t, 3682, ttt)

	ttt, err = issues_model.GetIssueTotalTrackedTime(db.DefaultContext, &issues_model.IssuesOptions{MilestoneIDs: []int64{1}}, optional.Some(true))
	require.NoError(t, err)
	assert.EqualValues(t, 0, ttt)

	ttt, err = issues_model.GetIssueTotalTrackedTime(db.DefaultContext, &issues_model.IssuesOptions{MilestoneIDs: []int64{1}}, optional.None[bool]())
	require.NoError(t, err)
	assert.EqualValues(t, 3682, ttt)
}
