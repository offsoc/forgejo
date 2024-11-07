// Copyright 2024 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"context"

	"code.gitea.io/gitea/models/db"
	api "code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/modules/timeutil"
)

func init() {
	db.RegisterModel(new(RepoLicense))
}

type RepoLicense struct { //revive:disable-line:exported
	ID          int64 `xorm:"pk autoincr"`
	RepoID      int64 `xorm:"UNIQUE(s) NOT NULL"`
	CommitID    string
	License     string             `xorm:"VARCHAR(255) UNIQUE(s) NOT NULL"`
	Path        string             `xorm:"UNIQUE(s) NOT NULL"`
	CreatedUnix timeutil.TimeStamp `xorm:"INDEX CREATED"`
	UpdatedUnix timeutil.TimeStamp `xorm:"INDEX UPDATED"`
}

// RepoLicenseList defines a list of repo licenses
type RepoLicenseList []*RepoLicense //revive:disable-line:exported

func (rll RepoLicenseList) StringList() []string {
	var licenses []string
	for _, rl := range rll {
		licenses = append(licenses, rl.License)
	}
	return licenses
}

// GetRepoLicenses returns the license statistics for a repository
func GetRepoLicenses(ctx context.Context, repo *Repository) (RepoLicenseList, error) {
	licenses := make(RepoLicenseList, 0)
	if err := db.GetEngine(ctx).Where("`repo_id` = ?", repo.ID).Asc("`license`").Find(&licenses); err != nil {
		return nil, err
	}
	return licenses, nil
}

// UpdateRepoLicenses updates the license statistics for repository
func UpdateRepoLicenses(ctx context.Context, repo *Repository, commitID string, licenses []*api.RepoLicense) error {
	oldLicenses, err := GetRepoLicenses(ctx, repo)
	if err != nil {
		return err
	}

	ctx, committer, err := db.TxContext(ctx)
	if err != nil {
		return err
	}
	defer committer.Close()

	for _, license := range licenses {
		upd := false
		for _, o := range oldLicenses {
			// Update already existing license
			if o.License == license.Name && o.Path == license.Path {
				_, err := db.GetEngine(ctx).Exec("UPDATE repo_license SET commit_id = ? WHERE id = ?", commitID, o.ID)
				if err != nil {
					return err
				}
				upd = true
				break
			}
		}

		_, err = db.GetEngine(ctx).Exec("DELETE FROM repo_license WHERE repo_id = ? AND license = ? AND commit_id != ?", repo.ID, license.Name, commitID)
		if err != nil {
			return err
		}

		// Insert new license
		if !upd {
			if err := db.Insert(ctx, &RepoLicense{
				RepoID:   repo.ID,
				CommitID: commitID,
				License:  license.Name,
				Path:     license.Path,
			}); err != nil {
				return err
			}
		}
	}

	_, err = db.GetEngine(ctx).Exec("DELETE FROM repo_license WHERE repo_id = ? AND commit_id != ?", repo.ID, commitID)
	if err != nil {
		return err
	}

	return committer.Commit()
}

// CopyLicense Copy originalRepo license information to destRepo (use for forked repo)
func CopyLicense(ctx context.Context, originalRepo, destRepo *Repository) error {
	repoLicenses, err := GetRepoLicenses(ctx, originalRepo)
	if err != nil {
		return err
	}
	if len(repoLicenses) > 0 {
		newRepoLicenses := make(RepoLicenseList, 0, len(repoLicenses))

		for _, rl := range repoLicenses {
			newRepoLicense := &RepoLicense{
				RepoID:   destRepo.ID,
				CommitID: rl.CommitID,
				License:  rl.License,
			}
			newRepoLicenses = append(newRepoLicenses, newRepoLicense)
		}
		if err := db.Insert(ctx, &newRepoLicenses); err != nil {
			return err
		}
	}
	return nil
}

// CleanRepoLicenses will remove all license record of the repo
func CleanRepoLicenses(ctx context.Context, repo *Repository) error {
	return db.DeleteBeans(ctx, &RepoLicense{
		RepoID: repo.ID,
	})
}
