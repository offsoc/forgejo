// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"testing"

	"forgejo.org/models/db"
	git_model "forgejo.org/models/git"
	"forgejo.org/models/unittest"
	"forgejo.org/modules/structs"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetLatestCommitStatusForPairs(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	t.Run("Empty", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		pairs, err := git_model.GetLatestCommitStatusForPairs(db.DefaultContext, nil)
		require.NoError(t, err)
		assert.Equal(t, map[int64][]*git_model.CommitStatus{}, pairs)
	})

	t.Run("Repo 1", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		pairs, err := git_model.GetLatestCommitStatusForPairs(db.DefaultContext, []git_model.RepoSHA{{RepoID: 1, SHA: "1234123412341234123412341234123412341234"}})
		require.NoError(t, err)

		assert.Equal(t, map[int64][]*git_model.CommitStatus{
			1: {
				{
					ID:          7,
					Index:       6,
					RepoID:      1,
					State:       structs.CommitStatusPending,
					SHA:         "1234123412341234123412341234123412341234",
					TargetURL:   "https://example.com/builds/",
					Description: "My awesome deploy service",
					ContextHash: "ae9547713a6665fc4261d0756904932085a41cf2",
					Context:     "deploy/awesomeness",
					CreatorID:   2,
				},
				{
					ID:          4,
					Index:       4,
					State:       structs.CommitStatusFailure,
					TargetURL:   "https://example.com/builds/",
					Description: "My awesome CI-service",
					Context:     "ci/awesomeness",
					CreatorID:   2,
					RepoID:      1,
					SHA:         "1234123412341234123412341234123412341234",
					ContextHash: "c65f4d64a3b14a3eced0c9b36799e66e1bd5ced7",
				},
				{
					ID:          3,
					Index:       3,
					State:       structs.CommitStatusSuccess,
					TargetURL:   "https://example.com/coverage/",
					Description: "My awesome Coverage service",
					Context:     "cov/awesomeness",
					CreatorID:   2,
					RepoID:      1,
					SHA:         "1234123412341234123412341234123412341234",
					ContextHash: "3929ac7bccd3fa1bf9b38ddedb77973b1b9a8cfe",
				},
			},
		}, pairs)
	})
	t.Run("Repo 62", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		pairs, err := git_model.GetLatestCommitStatusForPairs(db.DefaultContext, []git_model.RepoSHA{{RepoID: 62, SHA: "774f93df12d14931ea93259ae93418da4482fcc1"}})
		require.NoError(t, err)

		assert.Equal(t, map[int64][]*git_model.CommitStatus{
			62: {
				{
					ID:          8,
					Index:       2,
					RepoID:      62,
					State:       structs.CommitStatusError,
					TargetURL:   "/user2/test_workflows/actions",
					Description: "My awesome deploy service - v2",
					Context:     "deploy/awesomeness",
					SHA:         "774f93df12d14931ea93259ae93418da4482fcc1",
					ContextHash: "ae9547713a6665fc4261d0756904932085a41cf2",
					CreatorID:   2,
				},
			},
		}, pairs)
	})

	t.Run("Repo 62 non-existent sha", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		pairs, err := git_model.GetLatestCommitStatusForPairs(db.DefaultContext, []git_model.RepoSHA{{RepoID: 62, SHA: "774f93df12d14931ea93259ae93418da4482fcc"}})
		require.NoError(t, err)

		assert.Equal(t, map[int64][]*git_model.CommitStatus{}, pairs)
	})

	t.Run("SHA with non-existent repo id", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		pairs, err := git_model.GetLatestCommitStatusForPairs(db.DefaultContext, []git_model.RepoSHA{{RepoID: 1, SHA: "774f93df12d14931ea93259ae93418da4482fcc1"}})
		require.NoError(t, err)

		assert.Equal(t, map[int64][]*git_model.CommitStatus{}, pairs)
	})
}

func TestGetLatestCommitStatusForRepoCommitIDs(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	t.Run("Empty", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		repoStatuses, err := git_model.GetLatestCommitStatusForRepoCommitIDs(db.DefaultContext, 62, nil)
		require.NoError(t, err)
		assert.Equal(t, map[string][]*git_model.CommitStatus{}, repoStatuses)
	})

	t.Run("Repo 1", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		repoStatuses, err := git_model.GetLatestCommitStatusForRepoCommitIDs(db.DefaultContext, 1, []string{"1234123412341234123412341234123412341234"})
		require.NoError(t, err)
		assert.Equal(t, map[string][]*git_model.CommitStatus{
			"1234123412341234123412341234123412341234": {
				{
					ID:          3,
					Index:       3,
					State:       structs.CommitStatusSuccess,
					TargetURL:   "https://example.com/coverage/",
					Description: "My awesome Coverage service",
					Context:     "cov/awesomeness",
					CreatorID:   2,
					RepoID:      1,
					SHA:         "1234123412341234123412341234123412341234",
					ContextHash: "3929ac7bccd3fa1bf9b38ddedb77973b1b9a8cfe",
				},
				{
					ID:          4,
					Index:       4,
					State:       structs.CommitStatusFailure,
					TargetURL:   "https://example.com/builds/",
					Description: "My awesome CI-service",
					Context:     "ci/awesomeness",
					CreatorID:   2,
					RepoID:      1,
					SHA:         "1234123412341234123412341234123412341234",
					ContextHash: "c65f4d64a3b14a3eced0c9b36799e66e1bd5ced7",
				},
				{
					ID:          7,
					Index:       6,
					RepoID:      1,
					State:       structs.CommitStatusPending,
					SHA:         "1234123412341234123412341234123412341234",
					TargetURL:   "https://example.com/builds/",
					Description: "My awesome deploy service",
					ContextHash: "ae9547713a6665fc4261d0756904932085a41cf2",
					Context:     "deploy/awesomeness",
					CreatorID:   2,
				},
			},
		}, repoStatuses)
	})

	t.Run("Repo 62", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		repoStatuses, err := git_model.GetLatestCommitStatusForRepoCommitIDs(db.DefaultContext, 62, []string{"774f93df12d14931ea93259ae93418da4482fcc1"})
		require.NoError(t, err)
		assert.Equal(t, map[string][]*git_model.CommitStatus{
			"774f93df12d14931ea93259ae93418da4482fcc1": {
				{
					ID:          8,
					Index:       2,
					RepoID:      62,
					State:       structs.CommitStatusError,
					TargetURL:   "/user2/test_workflows/actions",
					Description: "My awesome deploy service - v2",
					Context:     "deploy/awesomeness",
					SHA:         "774f93df12d14931ea93259ae93418da4482fcc1",
					ContextHash: "ae9547713a6665fc4261d0756904932085a41cf2",
					CreatorID:   2,
				},
			},
		}, repoStatuses)
	})

	t.Run("Repo 62 non-existent sha", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		repoStatuses, err := git_model.GetLatestCommitStatusForRepoCommitIDs(db.DefaultContext, 62, []string{"774f93df12d14931ea93259ae93418da4482fcc"})
		require.NoError(t, err)
		assert.Equal(t, map[string][]*git_model.CommitStatus{}, repoStatuses)
	})

	t.Run("non-existent repo ID", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		repoStatuses, err := git_model.GetLatestCommitStatusForRepoCommitIDs(db.DefaultContext, 1, []string{"774f93df12d14931ea93259ae93418da4482fcc"})
		require.NoError(t, err)
		assert.Equal(t, map[string][]*git_model.CommitStatus{}, repoStatuses)
	})
}
