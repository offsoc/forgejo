// Copyright 2024 The Forgejo Authors c/o Codeberg e.V.. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"context"

	"forgejo.org/models/db"

	"xorm.io/builder"
)

// RepoFlag represents a single flag against a repository
type RepoFlag struct { //revive:disable-line:exported
	ID     int64  `xorm:"pk autoincr"`
	RepoID int64  `xorm:"UNIQUE(s) INDEX"`
	Name   string `xorm:"UNIQUE(s) INDEX"`
}

func init() {
	db.RegisterModel(new(RepoFlag))
}

// TableName provides the real table name
func (RepoFlag) TableName() string {
	return "forgejo_repo_flag"
}

// ListFlags returns the array of flags on the repo.
func (repo *Repository) ListFlags(ctx context.Context) ([]RepoFlag, error) {
	var flags []RepoFlag
	err := db.GetEngine(ctx).Table(&RepoFlag{}).Where("repo_id = ?", repo.ID).Find(&flags)
	if err != nil {
		return nil, err
	}
	return flags, nil
}

// IsFlagged returns whether a repo has any flags or not
func (repo *Repository) IsFlagged(ctx context.Context) bool {
	has, _ := db.Exist[RepoFlag](ctx, builder.Eq{"repo_id": repo.ID})
	return has
}

// GetFlag returns a single RepoFlag based on its name
func (repo *Repository) GetFlag(ctx context.Context, flagName string) (bool, *RepoFlag, error) {
	flag, has, err := db.Get[RepoFlag](ctx, builder.Eq{"repo_id": repo.ID, "name": flagName})
	if err != nil {
		return false, nil, err
	}
	return has, flag, nil
}

// HasFlag returns true if a repo has a given flag, false otherwise
func (repo *Repository) HasFlag(ctx context.Context, flagName string) bool {
	has, _ := db.Exist[RepoFlag](ctx, builder.Eq{"repo_id": repo.ID, "name": flagName})
	return has
}

// AddFlag adds a new flag to the repo
func (repo *Repository) AddFlag(ctx context.Context, flagName string) error {
	return db.Insert(ctx, RepoFlag{
		RepoID: repo.ID,
		Name:   flagName,
	})
}

// DeleteFlag removes a flag from the repo
func (repo *Repository) DeleteFlag(ctx context.Context, flagName string) (int64, error) {
	return db.DeleteByBean(ctx, &RepoFlag{RepoID: repo.ID, Name: flagName})
}

// ReplaceAllFlags replaces all flags of a repo with a new set
func (repo *Repository) ReplaceAllFlags(ctx context.Context, flagNames []string) error {
	ctx, committer, err := db.TxContext(ctx)
	if err != nil {
		return err
	}
	defer committer.Close()

	if err := db.DeleteBeans(ctx, &RepoFlag{RepoID: repo.ID}); err != nil {
		return err
	}

	if len(flagNames) == 0 {
		return committer.Commit()
	}

	var flags []RepoFlag
	for _, name := range flagNames {
		flags = append(flags, RepoFlag{
			RepoID: repo.ID,
			Name:   name,
		})
	}
	if err := db.Insert(ctx, &flags); err != nil {
		return err
	}

	return committer.Commit()
}
