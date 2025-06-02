// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package webhook

import (
	"testing"

	actions_model "forgejo.org/models/actions"
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
	api "forgejo.org/modules/structs"
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
	defer unittest.OverrideFixtures("services/webhook/TestPushCommits")()
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
	defer unittest.OverrideFixtures("services/webhook/TestPushCommits")()
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

func assertActionEqual(t *testing.T, expected_run *actions_model.ActionRun, actual_run *api.ActionRun) {
	assert.NotNil(t, expected_run)
	assert.NotNil(t, actual_run)
	// only test a few things
	assert.Equal(t, expected_run.ID, actual_run.ID)
	assert.Equal(t, expected_run.Status.String(), actual_run.Status)
	assert.Equal(t, expected_run.Index, actual_run.Index)
	assert.Equal(t, expected_run.RepoID, actual_run.Repo.ID)
	assert.Equal(t, expected_run.Stopped.AsTime(), actual_run.Stopped)
	assert.Equal(t, expected_run.Title, actual_run.Title)
	assert.Equal(t, expected_run.WorkflowID, actual_run.WorkflowID)
}

func TestAction(t *testing.T) {
	defer unittest.OverrideFixtures("services/webhook/TestPushCommits")()
	require.NoError(t, unittest.PrepareTestDatabase())

	triggerUser := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 2, OwnerID: triggerUser.ID})

	oldSuccessRun := &actions_model.ActionRun{
		ID:            1,
		Status:        actions_model.StatusSuccess,
		Index:         1,
		RepoID:        repo.ID,
		Stopped:       1693648027,
		WorkflowID:    "some_workflow",
		Title:         "oldSuccessRun",
		TriggerUser:   triggerUser,
		TriggerUserID: triggerUser.ID,
		TriggerEvent:  "push",
	}
	oldSuccessRun.LoadAttributes(db.DefaultContext)
	oldFailureRun := &actions_model.ActionRun{
		ID:            1,
		Status:        actions_model.StatusFailure,
		Index:         1,
		RepoID:        repo.ID,
		Stopped:       1693648027,
		WorkflowID:    "some_workflow",
		Title:         "oldFailureRun",
		TriggerUser:   triggerUser,
		TriggerUserID: triggerUser.ID,
		TriggerEvent:  "push",
	}
	oldFailureRun.LoadAttributes(db.DefaultContext)
	newSuccessRun := &actions_model.ActionRun{
		ID:            1,
		Status:        actions_model.StatusSuccess,
		Index:         1,
		RepoID:        repo.ID,
		Stopped:       1693648327,
		WorkflowID:    "some_workflow",
		Title:         "newSuccessRun",
		TriggerUser:   triggerUser,
		TriggerUserID: triggerUser.ID,
		TriggerEvent:  "push",
	}
	newSuccessRun.LoadAttributes(db.DefaultContext)
	newFailureRun := &actions_model.ActionRun{
		ID:            1,
		Status:        actions_model.StatusFailure,
		Index:         1,
		RepoID:        repo.ID,
		Stopped:       1693648327,
		WorkflowID:    "some_workflow",
		Title:         "newFailureRun",
		TriggerUser:   triggerUser,
		TriggerUserID: triggerUser.ID,
		TriggerEvent:  "push",
	}
	newFailureRun.LoadAttributes(db.DefaultContext)

	t.Run("Successful Run after Nothing", func(t *testing.T) {
		defer test.MockVariableValue(&setting.Webhook.PayloadCommitLimit, 10)()

		NewNotifier().ActionRunNowDone(db.DefaultContext, newSuccessRun, actions_model.StatusWaiting, nil)

		// there's only one of these at the time
		hookTask := unittest.AssertExistsAndLoadBean(t, &webhook_model.HookTask{}, unittest.Cond("event_type == 'action_run_hook_event_success' AND payload_content LIKE '%newSuccessRun%success%'"))

		var payloadContent structs.ActionPayload
		require.NoError(t, json.Unmarshal([]byte(hookTask.PayloadContent), &payloadContent))
		assert.Equal(t, api.ActionSuccess, payloadContent.Action)
		assert.Equal(t, actions_model.StatusWaiting.String(), payloadContent.PriorStatus)
		assertActionEqual(t, newSuccessRun, payloadContent.Run)
		assert.Nil(t, payloadContent.LastRun)
	})

	t.Run("Successful Run after Failure", func(t *testing.T) {
		defer test.MockVariableValue(&setting.Webhook.PayloadCommitLimit, 10)()

		NewNotifier().ActionRunNowDone(db.DefaultContext, newSuccessRun, actions_model.StatusWaiting, oldFailureRun)

		{
			hookTask := unittest.AssertExistsAndLoadBean(t, &webhook_model.HookTask{}, unittest.Cond("event_type == 'action_run_hook_event_success' AND payload_content LIKE '%newSuccessRun%success%oldFailureRun%'"))

			var payloadContent structs.ActionPayload
			require.NoError(t, json.Unmarshal([]byte(hookTask.PayloadContent), &payloadContent))
			assert.Equal(t, api.ActionSuccess, payloadContent.Action)
			assert.Equal(t, actions_model.StatusWaiting.String(), payloadContent.PriorStatus)
			assertActionEqual(t, newSuccessRun, payloadContent.Run)
			assertActionEqual(t, oldFailureRun, payloadContent.LastRun)
		}
		{
			hookTask := unittest.AssertExistsAndLoadBean(t, &webhook_model.HookTask{}, unittest.Cond("event_type == 'action_run_hook_event_recover' AND payload_content LIKE '%newSuccessRun%recovered%oldFailureRun%'"))

			var payloadContent structs.ActionPayload
			require.NoError(t, json.Unmarshal([]byte(hookTask.PayloadContent), &payloadContent))
			assert.Equal(t, api.ActionRecovered, payloadContent.Action)
			assert.Equal(t, actions_model.StatusWaiting.String(), payloadContent.PriorStatus)
			assertActionEqual(t, newSuccessRun, payloadContent.Run)
			assertActionEqual(t, oldFailureRun, payloadContent.LastRun)
		}
	})

	t.Run("Successful Run after Success", func(t *testing.T) {
		defer test.MockVariableValue(&setting.Webhook.PayloadCommitLimit, 10)()

		NewNotifier().ActionRunNowDone(db.DefaultContext, newSuccessRun, actions_model.StatusWaiting, oldSuccessRun)

		hookTask := unittest.AssertExistsAndLoadBean(t, &webhook_model.HookTask{}, unittest.Cond("event_type == 'action_run_hook_event_success' AND payload_content LIKE '%newSuccessRun%success%oldSuccessRun%'"))

		var payloadContent structs.ActionPayload
		require.NoError(t, json.Unmarshal([]byte(hookTask.PayloadContent), &payloadContent))
		assert.Equal(t, api.ActionSuccess, payloadContent.Action)
		assert.Equal(t, actions_model.StatusWaiting.String(), payloadContent.PriorStatus)
		assertActionEqual(t, newSuccessRun, payloadContent.Run)
		assertActionEqual(t, oldFailureRun, payloadContent.LastRun)
	})

	t.Run("Failed Run after Nothing", func(t *testing.T) {
		defer test.MockVariableValue(&setting.Webhook.PayloadCommitLimit, 10)()

		NewNotifier().ActionRunNowDone(db.DefaultContext, newFailureRun, actions_model.StatusWaiting, nil)

		// there should only be this one at the time
		hookTask := unittest.AssertExistsAndLoadBean(t, &webhook_model.HookTask{}, unittest.Cond("event_type == 'action_run_hook_event_failure' AND payload_content LIKE '%newFailureRun%failed%'"))

		var payloadContent structs.ActionPayload
		require.NoError(t, json.Unmarshal([]byte(hookTask.PayloadContent), &payloadContent))
		assert.Equal(t, api.ActionFailed, payloadContent.Action)
		assert.Equal(t, actions_model.StatusWaiting.String(), payloadContent.PriorStatus)
		assertActionEqual(t, newFailureRun, payloadContent.Run)
		assert.Nil(t, payloadContent.LastRun)
	})

	t.Run("Failed Run after Failure", func(t *testing.T) {
		defer test.MockVariableValue(&setting.Webhook.PayloadCommitLimit, 10)()

		NewNotifier().ActionRunNowDone(db.DefaultContext, newFailureRun, actions_model.StatusWaiting, oldFailureRun)

		hookTask := unittest.AssertExistsAndLoadBean(t, &webhook_model.HookTask{}, unittest.Cond("event_type == 'action_run_hook_event_failure' AND payload_content LIKE '%newFailureRun%failed%oldFailureRun%'"))

		var payloadContent structs.ActionPayload
		require.NoError(t, json.Unmarshal([]byte(hookTask.PayloadContent), &payloadContent))
		assert.Equal(t, api.ActionFailed, payloadContent.Action)
		assert.Equal(t, actions_model.StatusWaiting.String(), payloadContent.PriorStatus)
		assertActionEqual(t, newFailureRun, payloadContent.Run)
		assertActionEqual(t, oldFailureRun, payloadContent.LastRun)
	})

	t.Run("Failed Run after Success", func(t *testing.T) {
		defer test.MockVariableValue(&setting.Webhook.PayloadCommitLimit, 10)()

		NewNotifier().ActionRunNowDone(db.DefaultContext, newFailureRun, actions_model.StatusWaiting, oldSuccessRun)

		hookTask := unittest.AssertExistsAndLoadBean(t, &webhook_model.HookTask{}, unittest.Cond("event_type == 'action_run_hook_event_failure' AND payload_content LIKE '%newFailureRun%failed%oldSuccessRun%'"))

		var payloadContent structs.ActionPayload
		require.NoError(t, json.Unmarshal([]byte(hookTask.PayloadContent), &payloadContent))
		assert.Equal(t, api.ActionFailed, payloadContent.Action)
		assert.Equal(t, actions_model.StatusWaiting.String(), payloadContent.PriorStatus)
		assertActionEqual(t, newFailureRun, payloadContent.Run)
		assertActionEqual(t, oldSuccessRun, payloadContent.LastRun)
	})
}
