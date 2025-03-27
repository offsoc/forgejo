// Copyright 2024 The Forgejo Authors
// SPDX-License-Identifier: MIT

package actions

import (
	"testing"

	actions_model "forgejo.org/models/actions"
	"forgejo.org/models/db"
	"forgejo.org/models/unittest"
	webhook_module "forgejo.org/modules/webhook"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_SkipPullRequestEvent(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	repoID := int64(1)
	commitSHA := "1234"

	// event is not webhook_module.HookEventPullRequestSync, never skip
	assert.False(t, SkipPullRequestEvent(db.DefaultContext, webhook_module.HookEventPullRequest, repoID, commitSHA))

	// event is webhook_module.HookEventPullRequestSync but there is nothing in the ActionRun table, do not skip
	assert.False(t, SkipPullRequestEvent(db.DefaultContext, webhook_module.HookEventPullRequestSync, repoID, commitSHA))

	// there is a webhook_module.HookEventPullRequest event but the SHA is different, do not skip
	index := int64(1)
	run := &actions_model.ActionRun{
		Index:     index,
		Event:     webhook_module.HookEventPullRequest,
		RepoID:    repoID,
		CommitSHA: "othersha",
	}
	unittest.AssertSuccessfulInsert(t, run)
	assert.False(t, SkipPullRequestEvent(db.DefaultContext, webhook_module.HookEventPullRequestSync, repoID, commitSHA))

	// there already is a webhook_module.HookEventPullRequest with the same SHA, skip
	index++
	run = &actions_model.ActionRun{
		Index:     index,
		Event:     webhook_module.HookEventPullRequest,
		RepoID:    repoID,
		CommitSHA: commitSHA,
	}
	unittest.AssertSuccessfulInsert(t, run)
	assert.True(t, SkipPullRequestEvent(db.DefaultContext, webhook_module.HookEventPullRequestSync, repoID, commitSHA))
}
