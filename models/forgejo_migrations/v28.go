// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgejo_migrations //nolint:revive

import "xorm.io/xorm"

func AddHidePronounsOptionToUser(x *xorm.Engine) error {
	type User struct {
		ID                  int64 `xorm:"pk autoincr"`
		KeepPronounsPrivate bool  `xorm:"NOT NULL DEFAULT false"`
	}

	return x.Sync(&User{})
}
