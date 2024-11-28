// Copyright 2017 Gitea. All rights reserved.
// SPDX-License-Identifier: MIT

package git_test

import (
	"fmt"
	"testing"
	"time"

	actions_model "code.gitea.io/gitea/models/actions"
	"code.gitea.io/gitea/models/db"
	git_model "code.gitea.io/gitea/models/git"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/gitrepo"
	"code.gitea.io/gitea/modules/structs"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetCommitStatuses(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	repo1 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})

	sha1 := "1234123412341234123412341234123412341234"

	statuses, maxResults, err := db.FindAndCount[git_model.CommitStatus](db.DefaultContext, &git_model.CommitStatusOptions{
		ListOptions: db.ListOptions{Page: 1, PageSize: 50},
		RepoID:      repo1.ID,
		SHA:         sha1,
	})
	require.NoError(t, err)
	assert.EqualValues(t, 6, maxResults)
	assert.Len(t, statuses, 6)

	assert.Equal(t, "ci/awesomeness", statuses[0].Context)
	assert.Equal(t, structs.CommitStatusPending, statuses[0].State)
	assert.Equal(t, "https://try.gitea.io/api/v1/repos/user2/repo1/statuses/1234123412341234123412341234123412341234", statuses[0].APIURL(db.DefaultContext))

	assert.Equal(t, "cov/awesomeness", statuses[1].Context)
	assert.Equal(t, structs.CommitStatusWarning, statuses[1].State)
	assert.Equal(t, "https://try.gitea.io/api/v1/repos/user2/repo1/statuses/1234123412341234123412341234123412341234", statuses[1].APIURL(db.DefaultContext))

	assert.Equal(t, "cov/awesomeness", statuses[2].Context)
	assert.Equal(t, structs.CommitStatusSuccess, statuses[2].State)
	assert.Equal(t, "https://try.gitea.io/api/v1/repos/user2/repo1/statuses/1234123412341234123412341234123412341234", statuses[2].APIURL(db.DefaultContext))

	assert.Equal(t, "ci/awesomeness", statuses[3].Context)
	assert.Equal(t, structs.CommitStatusFailure, statuses[3].State)
	assert.Equal(t, "https://try.gitea.io/api/v1/repos/user2/repo1/statuses/1234123412341234123412341234123412341234", statuses[3].APIURL(db.DefaultContext))

	assert.Equal(t, "deploy/awesomeness", statuses[4].Context)
	assert.Equal(t, structs.CommitStatusError, statuses[4].State)
	assert.Equal(t, "https://try.gitea.io/api/v1/repos/user2/repo1/statuses/1234123412341234123412341234123412341234", statuses[4].APIURL(db.DefaultContext))

	assert.Equal(t, "deploy/awesomeness", statuses[5].Context)
	assert.Equal(t, structs.CommitStatusPending, statuses[5].State)
	assert.Equal(t, "https://try.gitea.io/api/v1/repos/user2/repo1/statuses/1234123412341234123412341234123412341234", statuses[5].APIURL(db.DefaultContext))

	statuses, maxResults, err = db.FindAndCount[git_model.CommitStatus](db.DefaultContext, &git_model.CommitStatusOptions{
		ListOptions: db.ListOptions{Page: 2, PageSize: 50},
		RepoID:      repo1.ID,
		SHA:         sha1,
	})
	require.NoError(t, err)
	assert.EqualValues(t, 6, maxResults)
	assert.Empty(t, statuses)
}

func Test_CalcCommitStatus(t *testing.T) {
	kases := []struct {
		statuses []*git_model.CommitStatus
		expected *git_model.CommitStatus
	}{
		{
			statuses: []*git_model.CommitStatus{
				{
					State: structs.CommitStatusPending,
				},
			},
			expected: &git_model.CommitStatus{
				State: structs.CommitStatusPending,
			},
		},
		{
			statuses: []*git_model.CommitStatus{
				{
					State: structs.CommitStatusSuccess,
				},
				{
					State: structs.CommitStatusPending,
				},
			},
			expected: &git_model.CommitStatus{
				State: structs.CommitStatusPending,
			},
		},
		{
			statuses: []*git_model.CommitStatus{
				{
					State: structs.CommitStatusSuccess,
				},
				{
					State: structs.CommitStatusPending,
				},
				{
					State: structs.CommitStatusSuccess,
				},
			},
			expected: &git_model.CommitStatus{
				State: structs.CommitStatusPending,
			},
		},
		{
			statuses: []*git_model.CommitStatus{
				{
					State: structs.CommitStatusError,
				},
				{
					State: structs.CommitStatusPending,
				},
				{
					State: structs.CommitStatusSuccess,
				},
			},
			expected: &git_model.CommitStatus{
				State: structs.CommitStatusError,
			},
		},
		{
			statuses: []*git_model.CommitStatus{
				{
					State: structs.CommitStatusWarning,
				},
				{
					State: structs.CommitStatusPending,
				},
				{
					State: structs.CommitStatusSuccess,
				},
			},
			expected: &git_model.CommitStatus{
				State: structs.CommitStatusWarning,
			},
		},
		{
			statuses: []*git_model.CommitStatus{
				{
					State: structs.CommitStatusSuccess,
					ID:    1,
				},
				{
					State: structs.CommitStatusSuccess,
					ID:    2,
				},
				{
					State: structs.CommitStatusSuccess,
					ID:    3,
				},
			},
			expected: &git_model.CommitStatus{
				State: structs.CommitStatusSuccess,
				ID:    3,
			},
		},
		{
			statuses: []*git_model.CommitStatus{
				{
					State: structs.CommitStatusFailure,
				},
				{
					State: structs.CommitStatusError,
				},
				{
					State: structs.CommitStatusWarning,
				},
			},
			expected: &git_model.CommitStatus{
				State: structs.CommitStatusError,
			},
		},
		{
			statuses: []*git_model.CommitStatus{},
			expected: nil,
		},
	}

	for _, kase := range kases {
		assert.Equal(t, kase.expected, git_model.CalcCommitStatus(kase.statuses))
	}
}

func TestFindRepoRecentCommitStatusContexts(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	repo2 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 2})
	user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	gitRepo, err := gitrepo.OpenRepository(git.DefaultContext, repo2)
	require.NoError(t, err)
	defer gitRepo.Close()

	commit, err := gitRepo.GetBranchCommit(repo2.DefaultBranch)
	require.NoError(t, err)

	defer func() {
		_, err := db.DeleteByBean(db.DefaultContext, &git_model.CommitStatus{
			RepoID:    repo2.ID,
			CreatorID: user2.ID,
			SHA:       commit.ID.String(),
		})
		require.NoError(t, err)
	}()

	err = git_model.NewCommitStatus(db.DefaultContext, git_model.NewCommitStatusOptions{
		Repo:    repo2,
		Creator: user2,
		SHA:     commit.ID,
		CommitStatus: &git_model.CommitStatus{
			State:     structs.CommitStatusFailure,
			TargetURL: "https://example.com/tests/",
			Context:   "compliance/lint-backend",
		},
	})
	require.NoError(t, err)

	err = git_model.NewCommitStatus(db.DefaultContext, git_model.NewCommitStatusOptions{
		Repo:    repo2,
		Creator: user2,
		SHA:     commit.ID,
		CommitStatus: &git_model.CommitStatus{
			State:     structs.CommitStatusSuccess,
			TargetURL: "https://example.com/tests/",
			Context:   "compliance/lint-backend",
		},
	})
	require.NoError(t, err)

	contexts, err := git_model.FindRepoRecentCommitStatusContexts(db.DefaultContext, repo2.ID, time.Hour)
	require.NoError(t, err)
	if assert.Len(t, contexts, 1) {
		assert.Equal(t, "compliance/lint-backend", contexts[0])
	}
}

func TestCommitStatusesHideActionsURL(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 4})
	run := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRun{ID: 791, RepoID: repo.ID})
	require.NoError(t, run.LoadAttributes(db.DefaultContext))

	statuses := []*git_model.CommitStatus{
		{
			RepoID:    repo.ID,
			TargetURL: fmt.Sprintf("%s/jobs/%d", run.Link(), run.Index),
		},
		{
			RepoID:    repo.ID,
			TargetURL: "https://mycicd.org/1",
		},
	}

	git_model.CommitStatusesHideActionsURL(db.DefaultContext, statuses)
	assert.Empty(t, statuses[0].TargetURL)
	assert.Equal(t, "https://mycicd.org/1", statuses[1].TargetURL)
}

func TestGetLatestCommitStatusForPairs(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	t.Run("All", func(t *testing.T) {
		pairs, err := git_model.GetLatestCommitStatusForPairs(db.DefaultContext, nil)
		require.NoError(t, err)

		assert.EqualValues(t, map[int64][]*git_model.CommitStatus{
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

	t.Run("Repo 1", func(t *testing.T) {
		pairs, err := git_model.GetLatestCommitStatusForPairs(db.DefaultContext, []git_model.RepoSHA{{1, "1234123412341234123412341234123412341234"}})
		require.NoError(t, err)

		assert.EqualValues(t, map[int64][]*git_model.CommitStatus{
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
		pairs, err := git_model.GetLatestCommitStatusForPairs(db.DefaultContext, []git_model.RepoSHA{{62, "774f93df12d14931ea93259ae93418da4482fcc1"}})
		require.NoError(t, err)

		assert.EqualValues(t, map[int64][]*git_model.CommitStatus{
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

	t.Run("Repo 62 nonexistant sha", func(t *testing.T) {
		pairs, err := git_model.GetLatestCommitStatusForPairs(db.DefaultContext, []git_model.RepoSHA{{62, "774f93df12d14931ea93259ae93418da4482fcc"}})
		require.NoError(t, err)

		assert.EqualValues(t, map[int64][]*git_model.CommitStatus{}, pairs)
	})

	t.Run("SHA with non existant repo id", func(t *testing.T) {
		pairs, err := git_model.GetLatestCommitStatusForPairs(db.DefaultContext, []git_model.RepoSHA{{1, "774f93df12d14931ea93259ae93418da4482fcc1"}})
		require.NoError(t, err)

		assert.EqualValues(t, map[int64][]*git_model.CommitStatus{}, pairs)
	})
}
