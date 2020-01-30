// Copyright 2014 The Gogs Authors. All rights reserved.
// Copyright 2019 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package models

import (
	"fmt"

	"code.gitea.io/gitea/modules/log"
)

// AccessMode specifies the users access mode
type AccessMode int

const (
	// AccessModeNone no access
	AccessModeNone AccessMode = iota // 0
	// AccessModeRead read access
	AccessModeRead // 1
	// AccessModeWrite write access
	AccessModeWrite // 2
	// AccessModeAdmin admin access
	AccessModeAdmin // 3
	// AccessModeOwner owner access
	AccessModeOwner // 4
)

func (mode AccessMode) String() string {
	switch mode {
	case AccessModeRead:
		return "read"
	case AccessModeWrite:
		return "write"
	case AccessModeAdmin:
		return "admin"
	case AccessModeOwner:
		return "owner"
	default:
		return "none"
	}
}

// ColorFormat provides a ColorFormatted version of this AccessMode
func (mode AccessMode) ColorFormat(s fmt.State) {
	log.ColorFprintf(s, "%d:%s",
		log.NewColoredIDValue(mode),
		mode)
}

// ParseAccessMode returns corresponding access mode to given permission string.
func ParseAccessMode(permission string) AccessMode {
	switch permission {
	case "write":
		return AccessModeWrite
	case "admin":
		return AccessModeAdmin
	default:
		return AccessModeRead
	}
}

// Access struct is deprecated
type Access struct {
	// FIXME: GAP: Remove Access from database

	ID     int64 `xorm:"pk autoincr"`
	UserID int64 `xorm:"UNIQUE(s)"`
	RepoID int64 `xorm:"UNIQUE(s)"`
	Mode   AccessMode
}

func accessLevel(e Engine, user *User, repo *Repository) (AccessMode, error) {
	mode := AccessModeNone
	var userID int64
	restricted := false

	if user != nil {
		userID = user.ID
		restricted = user.IsRestricted
	}

	if !restricted && !repo.IsPrivate {
		mode = AccessModeRead
	}

	if userID == 0 {
		return mode, nil
	}

	if userID == repo.OwnerID {
		return AccessModeOwner, nil
	}

	a := &Access{UserID: userID, RepoID: repo.ID}
	if has, err := e.Get(a); !has || err != nil {
		return mode, err
	}
	return a.Mode, nil
}

// GetRepositoryAccesses finds all repositories with their access mode where a user has any kind of access but does not own.
func (user *User) GetRepositoryAccesses() (map[*Repository]AccessMode, error) {
	// Xorm doesn't currently support such complex queries, so we first
	// retrieve the list of repositories; later we will retrieve the best
	// set of permissions for each and relate each other.
	rows, err := x.
		Where(accessibleRepositoryCondition(user)).
		And("repository.owner_id <> ?", user.ID).
		Rows(new(Repository))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var repos = make(map[*Repository]AccessMode, 10)
	var reposByID = make(map[int64]*Repository, 10)
	var ownerCache = make(map[int64]*User, 10)

	for rows.Next() {
		var repo Repository
		err = rows.Scan(&repo)
		if err != nil {
			return nil, err
		}

		var ok bool
		if repo.Owner, ok = ownerCache[repo.OwnerID]; !ok {
			if err = repo.GetOwner(); err != nil {
				return nil, err
			}
			ownerCache[repo.OwnerID] = repo.Owner
		}
		// Temporary nil permission until we find out the correct one
		repos[&repo] = AccessModeNone
		reposByID[repo.ID] = &repo
	}

	rows.Close()

	type bestAccessMode struct {
		RepoID		int64
		Mode		AccessMode
	}

	rows, err = x.Table("user_repo_unit").
		Join("INNER", "repository", "repository.id = user_repo_unit.repo_id").
		Where(accessibleRepositoryCondition(user)).
		And("repository.owner_id <> ?", user.ID).
		Select("repo_id, max(mode) as mode").
		GroupBy("repo_id").
		Rows(new(bestAccessMode))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	for rows.Next() {
		var bestMode bestAccessMode
		err = rows.Scan(&bestMode)
		if err != nil {
			return nil, err
		}
		if repo, ok := reposByID[bestMode.RepoID]; ok {
			repos[repo] = bestMode.Mode
		}
	}

	// Final pass, delete any repos from the map
	// that have not been updated (e.g. might have lose
	// access between the first and second queries)
	for _, repo := range reposByID {
		if repos[repo] == AccessModeNone {
			delete(repos, repo)
		}
	}

	return repos, nil
}

// GetAccessibleRepositories finds repositories which the user has access but does not own.
// If limit is smaller than 1 means returns all found results.
func (user *User) GetAccessibleRepositories(limit int) (repos []*Repository, _ error) {
	// FIXME: GAP: Test this query
	sess := x.Table(&Repository{}).
		Where(accessibleRepositoryCondition(user)).
		And("repository.owner_id <> ?", user.ID).
		Desc("updated_unix")
	if limit > 0 {
		sess.Limit(limit)
		repos = make([]*Repository, 0, limit)
	} else {
		repos = make([]*Repository, 0, 10)
	}
	
	return repos, sess.Find(&repos)
}

func maxAccessMode(modes ...AccessMode) AccessMode {
	max := AccessModeNone
	for _, mode := range modes {
		if mode > max {
			max = mode
		}
	}
	return max
}

// recalculateUserAccess recalculates repository access for a single user
func (repo *Repository) recalculateUserAccess(e Engine, uid int64) (err error) {
	return RebuildUserIDRepoUnits(e, uid, repo)
}

func (repo *Repository) recalculateAccesses(e Engine) error {
	return RebuildRepoUnits(e, repo, -1)
}

// RecalculateAccesses recalculates all accesses for repository.
func (repo *Repository) RecalculateAccesses() error {
	return repo.recalculateAccesses(x)
}

// addTeamAccesses adds accesses for a team on the repository.
func (repo *Repository) addTeamAccesses(e Engine, team *Team) error {
	return AddTeamRepoUnits(e, team, repo)
}

// addTeamAccesses adds accesses for a team on the repository.
func (repo *Repository) removeTeamAccesses(e Engine, teamID int64) error {
	return RebuildRepoUnits(e, repo, teamID)
}
