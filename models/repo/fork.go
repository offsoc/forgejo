// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"context"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/unit"
	user_model "code.gitea.io/gitea/models/user"

	"xorm.io/builder"
)

// GetRepositoriesByForkID returns all repositories with given fork ID.
func GetRepositoriesByForkID(ctx context.Context, forkID int64) ([]*Repository, error) {
	repos := make([]*Repository, 0, 10)
	return repos, db.GetEngine(ctx).
		Where("fork_id=?", forkID).
		Find(&repos)
}

// GetForkedRepo checks if given user has already forked a repository with given ID.
func GetForkedRepo(ctx context.Context, ownerID int64, baseRepo *Repository) *Repository {
	repo := new(Repository)
	query := db.GetEngine(ctx).Where("owner_id=?", ownerID)
	if baseRepo.IsFork {
		query = query.
			And("fork_id=? OR fork_id=?", baseRepo.ID, baseRepo.ForkID)
	} else {
		query = query.
			And("fork_id=?", baseRepo.ID)
	}
	has, _ := query.Get(repo)
	if has {
		return repo
	}
	return nil
}

// HasForkedRepo checks if given user has already forked a repository with given ID.
func HasForkedRepo(ctx context.Context, ownerID int64, baseRepo *Repository) bool {
	query := db.GetEngine(ctx).
		Table("repository").
		Where("owner_id=?", ownerID)
	if baseRepo.IsFork {
		query = query.And("fork_id=? OR fork_id=?", baseRepo.ID, baseRepo.ForkID)
	} else {
		query = query.And("fork_id=?", baseRepo.ID)
	}
	has, _ := query.Exist()
	return has
}

// GetUserFork return user forked repository from this repository, if not forked return nil.
// If that repository is itself a fork, the id of its base repository should be supplied,
// so that forks of this base repository are considered too.
func GetUserFork(ctx context.Context, repo *Repository, userID int64) (*Repository, error) {
	var forkedRepo Repository
	query := db.GetEngine(ctx).Where("owner_id = ?", userID)
	if repo.IsFork {
		query = query.And("fork_id = ? OR fork_id = ?", repo.ID, repo.ForkID)
	} else {
		query = query.And("fork_id = ?", repo.ID)
	}
	has, err := query.Get(&forkedRepo)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, nil
	}
	return &forkedRepo, nil
}

// GetForks returns all the forks of the repository that are visible to the user.
func GetForks(ctx context.Context, repo *Repository, user *user_model.User, listOptions db.ListOptions) ([]*Repository, int64, error) {
	sess := db.GetEngine(ctx).Where(AccessibleRepositoryCondition(user, unit.TypeInvalid))

	var forks []*Repository
	if listOptions.Page == 0 {
		forks = make([]*Repository, 0, repo.NumForks)
	} else {
		forks = make([]*Repository, 0, listOptions.PageSize)
		sess = db.SetSessionPagination(sess, &listOptions)
	}

	count, err := sess.FindAndCount(&forks, &Repository{ForkID: repo.ID})
	return forks, count, err
}

// IncrementRepoForkNum increment repository fork number
func IncrementRepoForkNum(ctx context.Context, repoID int64) error {
	_, err := db.GetEngine(ctx).Exec("UPDATE `repository` SET num_forks=num_forks+1 WHERE id=?", repoID)
	return err
}

// DecrementRepoForkNum decrement repository fork number
func DecrementRepoForkNum(ctx context.Context, repoID int64) error {
	_, err := db.GetEngine(ctx).Exec("UPDATE `repository` SET num_forks=num_forks-1 WHERE id=?", repoID)
	return err
}

// FindUserOrgForks returns the forked repositories for one user from a repository
func FindUserOrgForks(ctx context.Context, repoID, userID int64) ([]*Repository, error) {
	cond := builder.And(
		builder.Eq{"fork_id": repoID},
		builder.In("owner_id",
			builder.Select("org_id").
				From("org_user").
				Where(builder.Eq{"uid": userID}),
		),
	)

	var repos []*Repository
	return repos, db.GetEngine(ctx).Table("repository").Where(cond).Find(&repos)
}

// GetForksByUserAndOrgs return forked repos of the user and owned orgs
func GetForksByUserAndOrgs(ctx context.Context, user *user_model.User, repo *Repository) ([]*Repository, error) {
	var repoList []*Repository
	if user == nil {
		return repoList, nil
	}
	forkedRepo, err := GetUserFork(ctx, repo, user.ID)
	if err != nil {
		return repoList, err
	}
	if forkedRepo != nil {
		repoList = append(repoList, forkedRepo)
	}
	orgForks, err := FindUserOrgForks(ctx, repo.ID, user.ID)
	if err != nil {
		return nil, err
	}
	repoList = append(repoList, orgForks...)
	return repoList, nil
}
