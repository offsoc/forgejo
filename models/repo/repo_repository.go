// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT
package repo

import (
	"context"

	"forgejo.org/models/db"
	"forgejo.org/modules/validation"
)

func init() {
	db.RegisterModel(new(FollowingRepo))
}

func FindFollowingReposByRepoID(ctx context.Context, repoID int64) ([]*FollowingRepo, error) {
	maxFollowingRepos := 10
	sess := db.GetEngine(ctx).Where("repo_id=?", repoID)
	sess = sess.Limit(maxFollowingRepos, 0)
	followingRepoList := make([]*FollowingRepo, 0, maxFollowingRepos)
	err := sess.Find(&followingRepoList)
	if err != nil {
		return make([]*FollowingRepo, 0, maxFollowingRepos), err
	}
	for _, followingRepo := range followingRepoList {
		if res, err := validation.IsValid(*followingRepo); !res {
			return make([]*FollowingRepo, 0, maxFollowingRepos), err
		}
	}
	return followingRepoList, nil
}

func StoreFollowingRepos(ctx context.Context, localRepoID int64, followingRepoList []*FollowingRepo) error {
	for _, followingRepo := range followingRepoList {
		if res, err := validation.IsValid(*followingRepo); !res {
			return err
		}
	}

	// Begin transaction
	ctx, committer, err := db.TxContext((ctx))
	if err != nil {
		return err
	}
	defer committer.Close()

	_, err = db.GetEngine(ctx).Where("repo_id=?", localRepoID).Delete(FollowingRepo{})
	if err != nil {
		return err
	}
	for _, followingRepo := range followingRepoList {
		_, err = db.GetEngine(ctx).Insert(followingRepo)
		if err != nil {
			return err
		}
	}

	// Commit transaction
	return committer.Commit()
}
