// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repository

import (
	"context"

	"forgejo.org/models/repo"
	"forgejo.org/models/user"
	"forgejo.org/modules/setting"
	"forgejo.org/services/federation"
)

func StarRepoAndSendLikeActivities(ctx context.Context, doer user.User, repoID int64, star bool) error {
	if err := repo.StarRepo(ctx, doer.ID, repoID, star); err != nil {
		return err
	}

	if star && setting.Federation.Enabled {
		if err := federation.SendLikeActivities(ctx, doer, repoID); err != nil {
			return err
		}
	}

	return nil
}
