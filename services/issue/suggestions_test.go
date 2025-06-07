// Copyright 2025 The Gitea Authors. All rights reserved.
// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issue

import (
	"testing"

	"forgejo.org/models/db"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	"forgejo.org/modules/optional"
	"forgejo.org/modules/structs"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Suggestions(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	repo1 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})

	testCases := []struct {
		name               string
		isPull             optional.Option[bool]
		expectedSuggestion []*structs.IssueSuggestion
	}{
		{
			name: "All",
			expectedSuggestion: []*structs.IssueSuggestion{
				{
					Index:            5,
					State:            "open",
					Title:            "pull5",
					IsPr:             true,
					HasMerged:        false,
					IsWorkInProgress: false,
				},
				{
					Index:            1,
					State:            "open",
					Title:            "issue1",
					IsPr:             false,
					HasMerged:        false,
					IsWorkInProgress: false,
				},
				{
					Index:            4,
					State:            "closed",
					Title:            "issue5",
					IsPr:             false,
					HasMerged:        false,
					IsWorkInProgress: false,
				},
				{
					Index:            2,
					State:            "open",
					Title:            "issue2",
					IsPr:             true,
					HasMerged:        true,
					IsWorkInProgress: false,
				},
				{
					Index:            3,
					State:            "open",
					Title:            "issue3",
					IsPr:             true,
					HasMerged:        false,
					IsWorkInProgress: false,
				},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			suggestion, err := GetSuggestions(db.DefaultContext, repo1, testCase.isPull)
			require.NoError(t, err)
			assert.Equal(t, testCase.expectedSuggestion, suggestion)
		})
	}
}
