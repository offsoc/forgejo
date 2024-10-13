// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgejo_migrations //nolint:revive

import "xorm.io/xorm"

func AddRepoActivityPub(x *xorm.Engine) error {
	type Repository struct {
		ID                     int64  `xorm:"pk autoincr"`
		RepoActivityPubPrivPem string `xorm:"VARCHAR(1024)"`
		RepoActivityPubPubPem  string `xorm:"VARCHAR(1024)"`
	}

	return x.Sync(&Repository{})
}
