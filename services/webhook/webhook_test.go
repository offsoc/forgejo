// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package webhook

import (
	"fmt"
	"testing"

	"code.gitea.io/gitea/models/db"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/unittest"
	webhook_model "code.gitea.io/gitea/models/webhook"
	api "code.gitea.io/gitea/modules/structs"
	webhook_module "code.gitea.io/gitea/modules/webhook"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func activateWebhook(t *testing.T, hookID int64) {
	t.Helper()
	updated, err := db.GetEngine(db.DefaultContext).ID(hookID).Cols("is_active").Update(webhook_model.Webhook{IsActive: true})
	assert.Equal(t, int64(1), updated)
	require.NoError(t, err)
}

func TestPrepareWebhooks(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	activateWebhook(t, 1)

	hookTasks := []*webhook_model.HookTask{
		{HookID: 1, EventType: webhook_module.HookEventPush},
	}
	for _, hookTask := range hookTasks {
		unittest.AssertNotExistsBean(t, hookTask)
	}
	require.NoError(t, PrepareWebhooks(db.DefaultContext, EventSource{Repository: repo}, webhook_module.HookEventPush, &api.PushPayload{Commits: []*api.PayloadCommit{{}}}))
	for _, hookTask := range hookTasks {
		unittest.AssertExistsAndLoadBean(t, hookTask)
	}
}

func eventType(p api.Payloader) webhook_module.HookEventType {
	switch p.(type) {
	case *api.CreatePayload:
		return webhook_module.HookEventCreate
	case *api.DeletePayload:
		return webhook_module.HookEventDelete
	case *api.PushPayload:
		return webhook_module.HookEventPush
	}
	panic(fmt.Sprintf("no event type for payload %T", p))
}

func TestPrepareWebhooksBranchFilterMatch(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// branch_filter: {master,feature*}
	w := unittest.AssertExistsAndLoadBean(t, &webhook_model.Webhook{ID: 4})
	activateWebhook(t, w.ID)

	for _, p := range []api.Payloader{
		&api.PushPayload{Ref: "refs/heads/feature/7791"},
		&api.CreatePayload{Ref: "refs/heads/feature/7791"}, // branch creation
		&api.DeletePayload{Ref: "refs/heads/feature/7791"}, // branch deletion
	} {
		t.Run(fmt.Sprintf("%T", p), func(t *testing.T) {
			db.DeleteBeans(db.DefaultContext, webhook_model.HookTask{HookID: w.ID})
			typ := eventType(p)
			require.NoError(t, PrepareWebhook(db.DefaultContext, w, typ, p))
			unittest.AssertExistsAndLoadBean(t, &webhook_model.HookTask{
				HookID:    w.ID,
				EventType: typ,
			})
		})
	}
}

func TestPrepareWebhooksBranchFilterNoMatch(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// branch_filter: {master,feature*}
	w := unittest.AssertExistsAndLoadBean(t, &webhook_model.Webhook{ID: 4})
	activateWebhook(t, w.ID)

	for _, p := range []api.Payloader{
		&api.PushPayload{Ref: "refs/heads/fix_weird_bug"},
		&api.CreatePayload{Ref: "refs/heads/fix_weird_bug"}, // branch creation
		&api.DeletePayload{Ref: "refs/heads/fix_weird_bug"}, // branch deletion
	} {
		t.Run(fmt.Sprintf("%T", p), func(t *testing.T) {
			db.DeleteBeans(db.DefaultContext, webhook_model.HookTask{HookID: w.ID})
			require.NoError(t, PrepareWebhook(db.DefaultContext, w, eventType(p), p))
			unittest.AssertNotExistsBean(t, &webhook_model.HookTask{HookID: w.ID})
		})
	}
}
