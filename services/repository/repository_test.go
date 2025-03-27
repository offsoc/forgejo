// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repository

import (
	"testing"

	"forgejo.org/models/db"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unit"
	"forgejo.org/models/unittest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLinkedRepository(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	testCases := []struct {
		name             string
		attachID         int64
		expectedRepo     *repo_model.Repository
		expectedUnitType unit.Type
	}{
		{"LinkedIssue", 1, &repo_model.Repository{ID: 1}, unit.TypeIssues},
		{"LinkedComment", 3, &repo_model.Repository{ID: 1}, unit.TypePullRequests},
		{"LinkedRelease", 9, &repo_model.Repository{ID: 1}, unit.TypeReleases},
		{"Notlinked", 10, nil, -1},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			attach, err := repo_model.GetAttachmentByID(db.DefaultContext, tc.attachID)
			require.NoError(t, err)
			repo, unitType, err := LinkedRepository(db.DefaultContext, attach)
			require.NoError(t, err)
			if tc.expectedRepo != nil {
				assert.Equal(t, tc.expectedRepo.ID, repo.ID)
			}
			assert.Equal(t, tc.expectedUnitType, unitType)
		})
	}
}
