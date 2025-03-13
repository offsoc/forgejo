// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgejo_migrations //nolint:revive

import "xorm.io/xorm"

type FollowingRepo struct {
	ID               int64  `xorm:"pk autoincr"`
	RepoID           int64  `xorm:"UNIQUE(federation_repo_mapping) NOT NULL"`
	ExternalID       string `xorm:"UNIQUE(federation_repo_mapping) NOT NULL"`
	FederationHostID int64  `xorm:"UNIQUE(federation_repo_mapping) NOT NULL"`
	URI              string
}

func CreateFollowingRepoTable(x *xorm.Engine) error {
	return x.Sync(new(FollowingRepo))
}
