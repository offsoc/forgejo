// Copyright 2016 The Gogs Authors. All rights reserved.
// Copyright 2020 The Gitea Authors.
// SPDX-License-Identifier: MIT

package repository

import (
	"context"

	"forgejo.org/models"
	"forgejo.org/models/db"
	access_model "forgejo.org/models/perm/access"
	repo_model "forgejo.org/models/repo"
)

// DeleteCollaboration removes collaboration relation between the user and repository.
func DeleteCollaboration(ctx context.Context, repo *repo_model.Repository, uid int64) (err error) {
	collaboration := &repo_model.Collaboration{
		RepoID: repo.ID,
		UserID: uid,
	}

	ctx, committer, err := db.TxContext(ctx)
	if err != nil {
		return err
	}
	defer committer.Close()

	if has, err := db.GetEngine(ctx).Delete(collaboration); err != nil {
		return err
	} else if has == 0 {
		return committer.Commit()
	}
	if err = access_model.RecalculateAccesses(ctx, repo); err != nil {
		return err
	}

	if err = repo_model.WatchRepo(ctx, uid, repo.ID, false); err != nil {
		return err
	}

	if err = models.ReconsiderWatches(ctx, repo, uid); err != nil {
		return err
	}

	// Unassign a user from any issue (s)he has been assigned to in the repository
	if err := models.ReconsiderRepoIssuesAssignee(ctx, repo, uid); err != nil {
		return err
	}

	return committer.Commit()
}
