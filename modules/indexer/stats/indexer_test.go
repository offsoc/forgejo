// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package stats

import (
	"testing"
	"time"

	"code.gitea.io/gitea/models/db"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/unittest"
	"code.gitea.io/gitea/modules/queue"
	"code.gitea.io/gitea/modules/setting"

	_ "code.gitea.io/gitea/models"
	_ "code.gitea.io/gitea/models/actions"
	_ "code.gitea.io/gitea/models/activities"
	_ "code.gitea.io/gitea/models/forgefed"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	unittest.MainTest(m)
}

func TestRepoStatsIndex(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	setting.CfgProvider, _ = setting.NewConfigProviderFromData("")

	setting.LoadQueueSettings()

	err := Init()
	require.NoError(t, err)

	repo, err := repo_model.GetRepositoryByID(db.DefaultContext, 1)
	require.NoError(t, err)

	err = UpdateRepoIndexer(repo)
	require.NoError(t, err)

	require.NoError(t, queue.GetManager().FlushAll(t.Context(), 5*time.Second))

	status, err := repo_model.GetIndexerStatus(db.DefaultContext, repo, repo_model.RepoIndexerTypeStats)
	require.NoError(t, err)
	assert.Equal(t, "65f1bf27bc3bf70f64657658635e66094edbcb4d", status.CommitSha)
	langs, err := repo_model.GetTopLanguageStats(db.DefaultContext, repo, 5)
	require.NoError(t, err)
	assert.Empty(t, langs)
}
