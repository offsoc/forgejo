// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package project

import (
	"testing"

	"forgejo.org/models/db"
	"forgejo.org/models/unittest"
	"forgejo.org/modules/timeutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsProjectTypeValid(t *testing.T) {
	const UnknownType Type = 15

	cases := []struct {
		typ   Type
		valid bool
	}{
		{TypeIndividual, true},
		{TypeRepository, true},
		{TypeOrganization, true},
		{UnknownType, false},
	}

	for _, v := range cases {
		assert.Equal(t, v.valid, IsTypeValid(v.typ))
	}
}

func TestGetProjects(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	projects, err := db.Find[Project](db.DefaultContext, SearchOptions{RepoID: 1})
	require.NoError(t, err)

	// 1 value for this repo exists in the fixtures
	assert.Len(t, projects, 1)

	projects, err = db.Find[Project](db.DefaultContext, SearchOptions{RepoID: 3})
	require.NoError(t, err)

	// 1 value for this repo exists in the fixtures
	assert.Len(t, projects, 1)
}

func TestProject(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	project := &Project{
		Type:         TypeRepository,
		TemplateType: TemplateTypeBasicKanban,
		CardType:     CardTypeTextOnly,
		Title:        "New Project",
		RepoID:       1,
		CreatedUnix:  timeutil.TimeStampNow(),
		CreatorID:    2,
	}

	require.NoError(t, NewProject(db.DefaultContext, project))

	_, err := GetProjectByID(db.DefaultContext, project.ID)
	require.NoError(t, err)

	// Update project
	project.Title = "Updated title"
	require.NoError(t, UpdateProject(db.DefaultContext, project))

	projectFromDB, err := GetProjectByID(db.DefaultContext, project.ID)
	require.NoError(t, err)

	assert.Equal(t, project.Title, projectFromDB.Title)

	require.NoError(t, ChangeProjectStatusByRepoIDAndID(db.DefaultContext, project.RepoID, project.ID, true))

	// Retrieve from DB afresh to check if it is truly closed
	projectFromDB, err = GetProjectByID(db.DefaultContext, project.ID)
	require.NoError(t, err)

	assert.True(t, projectFromDB.IsClosed)
}

func TestProjectsSort(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	tests := []struct {
		sortType string
		wants    []int64
	}{
		{
			sortType: "default",
			wants:    []int64{1, 3, 2, 6, 5, 4},
		},
		{
			sortType: "oldest",
			wants:    []int64{4, 5, 6, 2, 3, 1},
		},
		{
			sortType: "recentupdate",
			wants:    []int64{1, 3, 2, 6, 5, 4},
		},
		{
			sortType: "leastupdate",
			wants:    []int64{4, 5, 6, 2, 3, 1},
		},
	}

	for _, tt := range tests {
		projects, count, err := db.FindAndCount[Project](db.DefaultContext, SearchOptions{
			OrderBy: GetSearchOrderByBySortType(tt.sortType),
		})
		require.NoError(t, err)
		assert.Equal(t, int64(6), count)
		if assert.Len(t, projects, 6) {
			for i := range projects {
				assert.Equal(t, tt.wants[i], projects[i].ID)
			}
		}
	}
}
