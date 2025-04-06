// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations //nolint:revive

import (
	"forgejo.org/modules/timeutil"

	"xorm.io/xorm"
)

func AddCreatedUnixToRedirect(x *xorm.Engine) error {
	type UserRedirect struct {
		ID          int64              `xorm:"pk autoincr"`
		CreatedUnix timeutil.TimeStamp `xorm:"created NOT NULL DEFAULT 0"`
	}
	return x.Sync(new(UserRedirect))
}
