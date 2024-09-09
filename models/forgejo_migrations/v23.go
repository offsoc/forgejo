// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgejo_migrations //nolint:revive

import (
	"code.gitea.io/gitea/modules/timeutil"

	"xorm.io/xorm"
)

func AddStarLists(x *xorm.Engine) error {
	type StarList struct {
		ID          int64  `xorm:"pk autoincr"`
		UserID      int64  `xorm:"INDEX UNIQUE(name)"`
		Name        string `xorm:"INDEX UNIQUE(name)"`
		Description string
		IsPrivate   bool
		CreatedUnix timeutil.TimeStamp `xorm:"INDEX created"`
	}

	err := x.Sync(&StarList{})
	if err != nil {
		return err
	}

	type StarListRepos struct {
		ID          int64              `xorm:"pk autoincr"`
		StarListID  int64              `xorm:"INDEX UNIQUE(repo)"`
		RepoID      int64              `xorm:"INDEX UNIQUE(repo)"`
		CreatedUnix timeutil.TimeStamp `xorm:"INDEX created"`
	}

	return x.Sync(&StarListRepos{})
}
