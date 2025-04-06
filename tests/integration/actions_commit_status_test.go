// Copyright 20124 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/url"
	"testing"

	actions_model "forgejo.org/models/actions"
	"forgejo.org/models/db"
	issues_model "forgejo.org/models/issues"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/test"
	"forgejo.org/services/actions"
	"forgejo.org/services/automerge"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActionsAutomerge(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		defer test.MockVariableValue(&setting.Actions.Enabled, true)()

		ctx := db.DefaultContext

		user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
		pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 2})
		job := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRunJob{ID: 292})

		assert.False(t, pr.HasMerged, "PR should not be merged")
		assert.Equal(t, issues_model.PullRequestStatusMergeable, pr.Status, "PR should be mergeable")

		scheduled, err := automerge.ScheduleAutoMerge(ctx, user, pr, repo_model.MergeStyleMerge, "Dummy", false)

		require.NoError(t, err, "PR should be scheduled for automerge")
		assert.True(t, scheduled, "PR should be scheduled for automerge")

		actions.CreateCommitStatus(ctx, job)

		pr = unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 2})

		assert.True(t, pr.HasMerged, "PR should be merged")
	},
	)
}
