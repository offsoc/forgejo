// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repository

import (
	"context"

	"forgejo.org/models/db"
	"forgejo.org/models/perm"
	access_model "forgejo.org/models/perm/access"
	repo_model "forgejo.org/models/repo"
	user_model "forgejo.org/models/user"

	"xorm.io/builder"
)

func AddCollaborator(ctx context.Context, repo *repo_model.Repository, u *user_model.User) error {
	if user_model.IsBlocked(ctx, repo.OwnerID, u.ID) || user_model.IsBlocked(ctx, u.ID, repo.OwnerID) {
		return user_model.ErrBlockedByUser
	}

	return db.WithTx(ctx, func(ctx context.Context) error {
		has, err := db.Exist[repo_model.Collaboration](ctx, builder.Eq{
			"repo_id": repo.ID,
			"user_id": u.ID,
		})
		if err != nil {
			return err
		} else if has {
			return nil
		}

		if err = db.Insert(ctx, &repo_model.Collaboration{
			RepoID: repo.ID,
			UserID: u.ID,
			Mode:   perm.AccessModeWrite,
		}); err != nil {
			return err
		}

		return access_model.RecalculateUserAccess(ctx, repo, u.ID)
	})
}
