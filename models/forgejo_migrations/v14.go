// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgejo_migrations //nolint:revive

import (
	"forgejo.org/models/migrations/base"

	"xorm.io/xorm"
)

func RemoveGiteaSpecificColumnsFromRepositoryAndBadge(x *xorm.Engine) error {
	// Make sure the columns exist before dropping them
	type Repository struct {
		ID                int64
		DefaultWikiBranch string
	}
	if err := x.Sync(&Repository{}); err != nil {
		return err
	}

	type Badge struct {
		ID   int64 `xorm:"pk autoincr"`
		Slug string
	}
	err := x.Sync(new(Badge))
	if err != nil {
		return err
	}

	sess := x.NewSession()
	defer sess.Close()
	if err := sess.Begin(); err != nil {
		return err
	}
	if err := base.DropTableColumns(sess, "repository", "default_wiki_branch"); err != nil {
		return err
	}
	if err := base.DropTableColumns(sess, "badge", "slug"); err != nil {
		return err
	}
	return sess.Commit()
}
