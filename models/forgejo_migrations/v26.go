// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgejo_migrations //nolint:revive

import "xorm.io/xorm"

func AddFediverseCreatorNameToUser(x *xorm.Engine) error {
	type User struct {
		ID                   int64  `xorm:"pk autoincr"`
		FediverseCreatorName string `xorm:"NOT NULL DEFAULT ''"`
	}

	return x.Sync(&User{})
}
