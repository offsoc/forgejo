// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package webhook

import (
	"context"
	"fmt"

	"forgejo.org/models/db"
)

// GetDefaultWebhooks returns all admin-default webhooks.
func GetDefaultWebhooks(ctx context.Context) ([]*Webhook, error) {
	return getAdminWebhooks(ctx, false)
}

// GetSystemOrDefaultWebhook returns admin system or default webhook by given ID.
func GetSystemOrDefaultWebhook(ctx context.Context, id int64) (*Webhook, error) {
	webhook := &Webhook{ID: id}
	has, err := db.GetEngine(ctx).
		Where("repo_id=? AND owner_id=?", 0, 0).
		Get(webhook)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, ErrWebhookNotExist{ID: id}
	}
	return webhook, nil
}

// GetSystemWebhooks returns all admin system webhooks.
func GetSystemWebhooks(ctx context.Context, onlyActive bool) ([]*Webhook, error) {
	return getAdminWebhooks(ctx, true, onlyActive)
}

func getAdminWebhooks(ctx context.Context, systemWebhooks bool, onlyActive ...bool) ([]*Webhook, error) {
	webhooks := make([]*Webhook, 0, 5)
	if len(onlyActive) > 0 && onlyActive[0] {
		return webhooks, db.GetEngine(ctx).
			Where("repo_id=? AND owner_id=? AND is_system_webhook=? AND is_active = ?", 0, 0, systemWebhooks, true).
			OrderBy("id").
			Find(&webhooks)
	}
	return webhooks, db.GetEngine(ctx).
		Where("repo_id=? AND owner_id=? AND is_system_webhook=?", 0, 0, systemWebhooks).
		OrderBy("id").
		Find(&webhooks)
}

// DeleteDefaultSystemWebhook deletes an admin-configured default or system webhook (where Org and Repo ID both 0)
func DeleteDefaultSystemWebhook(ctx context.Context, id int64) error {
	return db.WithTx(ctx, func(ctx context.Context) error {
		count, err := db.GetEngine(ctx).
			Where("repo_id=? AND owner_id=?", 0, 0).
			Delete(&Webhook{ID: id})
		if err != nil {
			return err
		} else if count == 0 {
			return ErrWebhookNotExist{ID: id}
		}

		_, err = db.DeleteByBean(ctx, &HookTask{HookID: id})
		return err
	})
}

// CopyDefaultWebhooksToRepo creates copies of the default webhooks in a new repo
func CopyDefaultWebhooksToRepo(ctx context.Context, repoID int64) error {
	ws, err := GetDefaultWebhooks(ctx)
	if err != nil {
		return fmt.Errorf("GetDefaultWebhooks: %v", err)
	}

	for _, w := range ws {
		w.ID = 0
		w.RepoID = repoID
		if err := CreateWebhook(ctx, w); err != nil {
			return fmt.Errorf("CreateWebhook: %v", err)
		}
	}
	return nil
}
