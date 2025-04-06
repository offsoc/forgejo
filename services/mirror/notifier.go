// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package mirror

import (
	"context"

	repo_model "forgejo.org/models/repo"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/repository"
	notify_service "forgejo.org/services/notify"
)

func init() {
	notify_service.RegisterNotifier(&mirrorNotifier{})
}

type mirrorNotifier struct {
	notify_service.NullNotifier
}

var _ notify_service.Notifier = &mirrorNotifier{}

func (m *mirrorNotifier) PushCommits(ctx context.Context, _ *user_model.User, repo *repo_model.Repository, _ *repository.PushUpdateOptions, _ *repository.PushCommits) {
	syncPushMirrorWithSyncOnCommit(ctx, repo.ID)
}

func (m *mirrorNotifier) SyncPushCommits(ctx context.Context, _ *user_model.User, repo *repo_model.Repository, _ *repository.PushUpdateOptions, _ *repository.PushCommits) {
	syncPushMirrorWithSyncOnCommit(ctx, repo.ID)
}
