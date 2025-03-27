// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package v1_22 //nolint

import (
	"forgejo.org/modules/timeutil"

	"xorm.io/xorm"
)

func AddCreatedToIssue(x *xorm.Engine) error {
	type Issue struct {
		ID      int64 `xorm:"pk autoincr"`
		Created timeutil.TimeStampNano
	}

	return x.Sync(&Issue{})
}
