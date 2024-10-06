// Copyright 20124 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/url"
	"testing"

	actions_model "code.gitea.io/gitea/models/actions"
	"code.gitea.io/gitea/models/db"
	issues_model "code.gitea.io/gitea/models/issues"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/services/actions"
	"code.gitea.io/gitea/services/automerge"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActionsAutomerge(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		assert.True(t, setting.Actions.Enabled, "Actions should be enabled")

		ctx := db.DefaultContext

		user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
		pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 2})
		job := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRunJob{ID: 292})

		assert.False(t, pr.HasMerged, "PR should not be merged")
		assert.Equal(t, issues_model.PullRequestStatusMergeable, pr.Status, "PR should be mergeable")

		scheduled, err := automerge.ScheduleAutoMerge(ctx, user, pr, repo_model.MergeStyleMerge, "Dummy")

		require.NoError(t, err, "PR should be scheduled for automerge")
		assert.True(t, scheduled, "PR should be scheduled for automerge")

		actions.CreateCommitStatus(ctx, job)

		pr = unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 2})

		assert.True(t, pr.HasMerged, "PR should be merged")
	},
	)
}
