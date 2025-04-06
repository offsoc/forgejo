// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issues_test

import (
	"sort"
	"testing"

	"forgejo.org/models/db"
	issues_model "forgejo.org/models/issues"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	"forgejo.org/modules/optional"
	"forgejo.org/modules/setting"
	api "forgejo.org/modules/structs"
	"forgejo.org/modules/timeutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMilestone_State(t *testing.T) {
	assert.Equal(t, api.StateOpen, (&issues_model.Milestone{IsClosed: false}).State())
	assert.Equal(t, api.StateClosed, (&issues_model.Milestone{IsClosed: true}).State())
}

func TestGetMilestoneByRepoID(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	milestone, err := issues_model.GetMilestoneByRepoID(db.DefaultContext, 1, 1)
	require.NoError(t, err)
	assert.EqualValues(t, 1, milestone.ID)
	assert.EqualValues(t, 1, milestone.RepoID)

	_, err = issues_model.GetMilestoneByRepoID(db.DefaultContext, unittest.NonexistentID, unittest.NonexistentID)
	assert.True(t, issues_model.IsErrMilestoneNotExist(err))
}

func TestGetMilestonesByRepoID(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	test := func(repoID int64, state api.StateType) {
		var isClosed optional.Option[bool]
		switch state {
		case api.StateClosed, api.StateOpen:
			isClosed = optional.Some(state == api.StateClosed)
		}
		repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: repoID})
		milestones, err := db.Find[issues_model.Milestone](db.DefaultContext, issues_model.FindMilestoneOptions{
			RepoID:   repo.ID,
			IsClosed: isClosed,
		})
		require.NoError(t, err)

		var n int

		switch state {
		case api.StateClosed:
			n = repo.NumClosedMilestones

		case api.StateAll:
			n = repo.NumMilestones

		case api.StateOpen:
			fallthrough

		default:
			n = repo.NumOpenMilestones
		}

		assert.Len(t, milestones, n)
		for _, milestone := range milestones {
			assert.Equal(t, repoID, milestone.RepoID)
		}
	}
	test(1, api.StateOpen)
	test(1, api.StateAll)
	test(1, api.StateClosed)
	test(2, api.StateOpen)
	test(2, api.StateAll)
	test(2, api.StateClosed)
	test(3, api.StateOpen)
	test(3, api.StateClosed)
	test(3, api.StateAll)

	milestones, err := db.Find[issues_model.Milestone](db.DefaultContext, issues_model.FindMilestoneOptions{
		RepoID:   unittest.NonexistentID,
		IsClosed: optional.Some(false),
	})
	require.NoError(t, err)
	assert.Empty(t, milestones)
}

func TestGetMilestones(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	test := func(sortType string, sortCond func(*issues_model.Milestone) int) {
		for _, page := range []int{0, 1} {
			milestones, err := db.Find[issues_model.Milestone](db.DefaultContext, issues_model.FindMilestoneOptions{
				ListOptions: db.ListOptions{
					Page:     page,
					PageSize: setting.UI.IssuePagingNum,
				},
				RepoID:   repo.ID,
				IsClosed: optional.Some(false),
				SortType: sortType,
			})
			require.NoError(t, err)
			assert.Len(t, milestones, repo.NumMilestones-repo.NumClosedMilestones)
			values := make([]int, len(milestones))
			for i, milestone := range milestones {
				values[i] = sortCond(milestone)
			}
			assert.True(t, sort.IntsAreSorted(values))

			milestones, err = db.Find[issues_model.Milestone](db.DefaultContext, issues_model.FindMilestoneOptions{
				ListOptions: db.ListOptions{
					Page:     page,
					PageSize: setting.UI.IssuePagingNum,
				},
				RepoID:   repo.ID,
				IsClosed: optional.Some(true),
				Name:     "",
				SortType: sortType,
			})
			require.NoError(t, err)
			assert.Len(t, milestones, repo.NumClosedMilestones)
			values = make([]int, len(milestones))
			for i, milestone := range milestones {
				values[i] = sortCond(milestone)
			}
			assert.True(t, sort.IntsAreSorted(values))
		}
	}
	test("furthestduedate", func(milestone *issues_model.Milestone) int {
		return -int(milestone.DeadlineUnix)
	})
	test("leastcomplete", func(milestone *issues_model.Milestone) int {
		return milestone.Completeness
	})
	test("mostcomplete", func(milestone *issues_model.Milestone) int {
		return -milestone.Completeness
	})
	test("leastissues", func(milestone *issues_model.Milestone) int {
		return milestone.NumIssues
	})
	test("mostissues", func(milestone *issues_model.Milestone) int {
		return -milestone.NumIssues
	})
	test("soonestduedate", func(milestone *issues_model.Milestone) int {
		return int(milestone.DeadlineUnix)
	})
}

func TestCountRepoMilestones(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	test := func(repoID int64) {
		repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: repoID})
		count, err := db.Count[issues_model.Milestone](db.DefaultContext, issues_model.FindMilestoneOptions{
			RepoID: repoID,
		})
		require.NoError(t, err)
		assert.EqualValues(t, repo.NumMilestones, count)
	}
	test(1)
	test(2)
	test(3)

	count, err := db.Count[issues_model.Milestone](db.DefaultContext, issues_model.FindMilestoneOptions{
		RepoID: unittest.NonexistentID,
	})
	require.NoError(t, err)
	assert.EqualValues(t, 0, count)
}

func TestCountRepoClosedMilestones(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	test := func(repoID int64) {
		repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: repoID})
		count, err := db.Count[issues_model.Milestone](db.DefaultContext, issues_model.FindMilestoneOptions{
			RepoID:   repoID,
			IsClosed: optional.Some(true),
		})
		require.NoError(t, err)
		assert.EqualValues(t, repo.NumClosedMilestones, count)
	}
	test(1)
	test(2)
	test(3)

	count, err := db.Count[issues_model.Milestone](db.DefaultContext, issues_model.FindMilestoneOptions{
		RepoID:   unittest.NonexistentID,
		IsClosed: optional.Some(true),
	})
	require.NoError(t, err)
	assert.EqualValues(t, 0, count)
}

func TestCountMilestonesByRepoIDs(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	milestonesCount := func(repoID int64) (int, int) {
		repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: repoID})
		return repo.NumOpenMilestones, repo.NumClosedMilestones
	}
	repo1OpenCount, repo1ClosedCount := milestonesCount(1)
	repo2OpenCount, repo2ClosedCount := milestonesCount(2)

	openCounts, err := issues_model.CountMilestonesMap(db.DefaultContext, issues_model.FindMilestoneOptions{
		RepoIDs:  []int64{1, 2},
		IsClosed: optional.Some(false),
	})
	require.NoError(t, err)
	assert.EqualValues(t, repo1OpenCount, openCounts[1])
	assert.EqualValues(t, repo2OpenCount, openCounts[2])

	closedCounts, err := issues_model.CountMilestonesMap(db.DefaultContext,
		issues_model.FindMilestoneOptions{
			RepoIDs:  []int64{1, 2},
			IsClosed: optional.Some(true),
		})
	require.NoError(t, err)
	assert.EqualValues(t, repo1ClosedCount, closedCounts[1])
	assert.EqualValues(t, repo2ClosedCount, closedCounts[2])
}

func TestGetMilestonesByRepoIDs(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	repo1 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	repo2 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 2})
	test := func(sortType string, sortCond func(*issues_model.Milestone) int) {
		for _, page := range []int{0, 1} {
			openMilestones, err := db.Find[issues_model.Milestone](db.DefaultContext, issues_model.FindMilestoneOptions{
				ListOptions: db.ListOptions{
					Page:     page,
					PageSize: setting.UI.IssuePagingNum,
				},
				RepoIDs:  []int64{repo1.ID, repo2.ID},
				IsClosed: optional.Some(false),
				SortType: sortType,
			})
			require.NoError(t, err)
			assert.Len(t, openMilestones, repo1.NumOpenMilestones+repo2.NumOpenMilestones)
			values := make([]int, len(openMilestones))
			for i, milestone := range openMilestones {
				values[i] = sortCond(milestone)
			}
			assert.True(t, sort.IntsAreSorted(values))

			closedMilestones, err := db.Find[issues_model.Milestone](db.DefaultContext,
				issues_model.FindMilestoneOptions{
					ListOptions: db.ListOptions{
						Page:     page,
						PageSize: setting.UI.IssuePagingNum,
					},
					RepoIDs:  []int64{repo1.ID, repo2.ID},
					IsClosed: optional.Some(true),
					SortType: sortType,
				})
			require.NoError(t, err)
			assert.Len(t, closedMilestones, repo1.NumClosedMilestones+repo2.NumClosedMilestones)
			values = make([]int, len(closedMilestones))
			for i, milestone := range closedMilestones {
				values[i] = sortCond(milestone)
			}
			assert.True(t, sort.IntsAreSorted(values))
		}
	}
	test("furthestduedate", func(milestone *issues_model.Milestone) int {
		return -int(milestone.DeadlineUnix)
	})
	test("leastcomplete", func(milestone *issues_model.Milestone) int {
		return milestone.Completeness
	})
	test("mostcomplete", func(milestone *issues_model.Milestone) int {
		return -milestone.Completeness
	})
	test("leastissues", func(milestone *issues_model.Milestone) int {
		return milestone.NumIssues
	})
	test("mostissues", func(milestone *issues_model.Milestone) int {
		return -milestone.NumIssues
	})
	test("soonestduedate", func(milestone *issues_model.Milestone) int {
		return int(milestone.DeadlineUnix)
	})
}

func TestNewMilestone(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	milestone := &issues_model.Milestone{
		RepoID:  1,
		Name:    "milestoneName",
		Content: "milestoneContent",
	}

	require.NoError(t, issues_model.NewMilestone(db.DefaultContext, milestone))
	unittest.AssertExistsAndLoadBean(t, milestone)
	unittest.CheckConsistencyFor(t, &repo_model.Repository{ID: milestone.RepoID}, &issues_model.Milestone{})
}

func TestChangeMilestoneStatusByRepoIDAndID(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	require.NoError(t, issues_model.ChangeMilestoneStatusByRepoIDAndID(db.DefaultContext, 1, 1, true))
	unittest.AssertExistsAndLoadBean(t, &issues_model.Milestone{ID: 1, IsClosed: true})
	unittest.CheckConsistencyFor(t, &repo_model.Repository{ID: 1}, &issues_model.Milestone{})

	require.NoError(t, issues_model.ChangeMilestoneStatusByRepoIDAndID(db.DefaultContext, 1, 1, false))
	unittest.AssertExistsAndLoadBean(t, &issues_model.Milestone{ID: 1}, "is_closed=0")
	unittest.CheckConsistencyFor(t, &repo_model.Repository{ID: 1}, &issues_model.Milestone{})
}

func TestDeleteMilestoneByRepoID(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	require.NoError(t, issues_model.DeleteMilestoneByRepoID(db.DefaultContext, 1, 1))
	unittest.AssertNotExistsBean(t, &issues_model.Milestone{ID: 1})
	unittest.CheckConsistencyFor(t, &repo_model.Repository{ID: 1})

	require.NoError(t, issues_model.DeleteMilestoneByRepoID(db.DefaultContext, unittest.NonexistentID, unittest.NonexistentID))
}

func TestUpdateMilestone(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	milestone := unittest.AssertExistsAndLoadBean(t, &issues_model.Milestone{ID: 1})
	milestone.Name = " newMilestoneName  "
	milestone.Content = "newMilestoneContent"
	require.NoError(t, issues_model.UpdateMilestone(db.DefaultContext, milestone, milestone.IsClosed))
	milestone = unittest.AssertExistsAndLoadBean(t, &issues_model.Milestone{ID: 1})
	assert.Equal(t, "newMilestoneName", milestone.Name)
	unittest.CheckConsistencyFor(t, &issues_model.Milestone{})
}

func TestUpdateMilestoneCounters(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{MilestoneID: 1},
		"is_closed=0")

	issue.IsClosed = true
	issue.ClosedUnix = timeutil.TimeStampNow()
	_, err := db.GetEngine(db.DefaultContext).ID(issue.ID).Cols("is_closed", "closed_unix").Update(issue)
	require.NoError(t, err)
	require.NoError(t, issues_model.UpdateMilestoneCounters(db.DefaultContext, issue.MilestoneID))
	unittest.CheckConsistencyFor(t, &issues_model.Milestone{})

	issue.IsClosed = false
	issue.ClosedUnix = 0
	_, err = db.GetEngine(db.DefaultContext).ID(issue.ID).Cols("is_closed", "closed_unix").Update(issue)
	require.NoError(t, err)
	require.NoError(t, issues_model.UpdateMilestoneCounters(db.DefaultContext, issue.MilestoneID))
	unittest.CheckConsistencyFor(t, &issues_model.Milestone{})
}

func TestMigrate_InsertMilestones(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	reponame := "repo1"
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{Name: reponame})
	name := "milestonetest1"
	ms := &issues_model.Milestone{
		RepoID: repo.ID,
		Name:   name,
	}
	err := issues_model.InsertMilestones(db.DefaultContext, ms)
	require.NoError(t, err)
	unittest.AssertExistsAndLoadBean(t, ms)
	repoModified := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: repo.ID})
	assert.Equal(t, repo.NumMilestones+1, repoModified.NumMilestones)

	unittest.CheckConsistencyFor(t, &issues_model.Milestone{})
}
