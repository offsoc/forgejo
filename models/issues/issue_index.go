// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issues

import (
	"context"

	"forgejo.org/models/db"
)

func GetMaxIssueIndexForRepo(ctx context.Context, repoID int64) (int64, error) {
	var max int64
	if _, err := db.GetEngine(ctx).Select("MAX(`index`)").Table("issue").Where("repo_id=?", repoID).Get(&max); err != nil {
		return 0, err
	}
	return max, nil
}

// RecalculateIssueIndexForRepo create issue_index for repo if not exist and
// update it based on highest index of existing issues assigned to a repo
func RecalculateIssueIndexForRepo(ctx context.Context, repoID int64) error {
	ctx, committer, err := db.TxContext(ctx)
	if err != nil {
		return err
	}
	defer committer.Close()

	max, err := GetMaxIssueIndexForRepo(ctx, repoID)
	if err != nil {
		return err
	}

	if err = db.SyncMaxResourceIndex(ctx, "issue_index", repoID, max); err != nil {
		return err
	}

	return committer.Commit()
}
