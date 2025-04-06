// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package webhook

import (
	"path/filepath"
	"testing"

	"forgejo.org/models/db"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	webhook_model "forgejo.org/models/webhook"
	"forgejo.org/modules/git"
	"forgejo.org/modules/json"
	"forgejo.org/modules/repository"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/structs"
	"forgejo.org/modules/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func pushCommits() *repository.PushCommits {
	pushCommits := repository.NewPushCommits()
	pushCommits.Commits = []*repository.PushCommit{
		{
			Sha1:           "2c54faec6c45d31c1abfaecdab471eac6633738a",
			CommitterEmail: "user2@example.com",
			CommitterName:  "User2",
			AuthorEmail:    "user2@example.com",
			AuthorName:     "User2",
			Message:        "not signed commit",
		},
		{
			Sha1:           "205ac761f3326a7ebe416e8673760016450b5cec",
			CommitterEmail: "user2@example.com",
			CommitterName:  "User2",
			AuthorEmail:    "user2@example.com",
			AuthorName:     "User2",
			Message:        "good signed commit (with not yet validated email)",
		},
		{
			Sha1:           "1032bbf17fbc0d9c95bb5418dabe8f8c99278700",
			CommitterEmail: "user2@example.com",
			CommitterName:  "User2",
			AuthorEmail:    "user2@example.com",
			AuthorName:     "User2",
			Message:        "good signed commit",
		},
	}
	pushCommits.HeadCommit = &repository.PushCommit{Sha1: "2c54faec6c45d31c1abfaecdab471eac6633738a"}
	return pushCommits
}

func TestSyncPushCommits(t *testing.T) {
	defer unittest.OverrideFixtures(
		unittest.FixturesOptions{
			Dir:  filepath.Join(setting.AppWorkPath, "models/fixtures/"),
			Base: setting.AppWorkPath,
			Dirs: []string{"services/webhook/TestPushCommits"},
		},
	)()
	require.NoError(t, unittest.PrepareTestDatabase())

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 2, OwnerID: user.ID})

	t.Run("All commits", func(t *testing.T) {
		defer test.MockVariableValue(&setting.Webhook.PayloadCommitLimit, 10)()

		NewNotifier().SyncPushCommits(db.DefaultContext, user, repo, &repository.PushUpdateOptions{RefFullName: git.RefNameFromBranch("master-1")}, pushCommits())

		hookTask := unittest.AssertExistsAndLoadBean(t, &webhook_model.HookTask{}, unittest.Cond("payload_content LIKE '%master-1%'"))

		var payloadContent structs.PushPayload
		require.NoError(t, json.Unmarshal([]byte(hookTask.PayloadContent), &payloadContent))
		assert.Len(t, payloadContent.Commits, 3)
	})

	t.Run("Only one commit", func(t *testing.T) {
		defer test.MockVariableValue(&setting.Webhook.PayloadCommitLimit, 1)()

		NewNotifier().SyncPushCommits(db.DefaultContext, user, repo, &repository.PushUpdateOptions{RefFullName: git.RefNameFromBranch("main-1")}, pushCommits())

		hookTask := unittest.AssertExistsAndLoadBean(t, &webhook_model.HookTask{}, unittest.Cond("payload_content LIKE '%main-1%'"))

		var payloadContent structs.PushPayload
		require.NoError(t, json.Unmarshal([]byte(hookTask.PayloadContent), &payloadContent))
		assert.Len(t, payloadContent.Commits, 1)
		assert.Equal(t, "2c54faec6c45d31c1abfaecdab471eac6633738a", payloadContent.Commits[0].ID)
	})
}

func TestPushCommits(t *testing.T) {
	defer unittest.OverrideFixtures(
		unittest.FixturesOptions{
			Dir:  filepath.Join(setting.AppWorkPath, "models/fixtures/"),
			Base: setting.AppWorkPath,
			Dirs: []string{"services/webhook/TestPushCommits"},
		},
	)()
	require.NoError(t, unittest.PrepareTestDatabase())

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 2, OwnerID: user.ID})

	t.Run("All commits", func(t *testing.T) {
		defer test.MockVariableValue(&setting.Webhook.PayloadCommitLimit, 10)()

		NewNotifier().PushCommits(db.DefaultContext, user, repo, &repository.PushUpdateOptions{RefFullName: git.RefNameFromBranch("master-2")}, pushCommits())

		hookTask := unittest.AssertExistsAndLoadBean(t, &webhook_model.HookTask{}, unittest.Cond("payload_content LIKE '%master-2%'"))

		var payloadContent structs.PushPayload
		require.NoError(t, json.Unmarshal([]byte(hookTask.PayloadContent), &payloadContent))
		assert.Len(t, payloadContent.Commits, 3)
	})

	t.Run("Only one commit", func(t *testing.T) {
		defer test.MockVariableValue(&setting.Webhook.PayloadCommitLimit, 1)()

		NewNotifier().PushCommits(db.DefaultContext, user, repo, &repository.PushUpdateOptions{RefFullName: git.RefNameFromBranch("main-2")}, pushCommits())

		hookTask := unittest.AssertExistsAndLoadBean(t, &webhook_model.HookTask{}, unittest.Cond("payload_content LIKE '%main-2%'"))

		var payloadContent structs.PushPayload
		require.NoError(t, json.Unmarshal([]byte(hookTask.PayloadContent), &payloadContent))
		assert.Len(t, payloadContent.Commits, 1)
		assert.Equal(t, "2c54faec6c45d31c1abfaecdab471eac6633738a", payloadContent.Commits[0].ID)
	})
}
