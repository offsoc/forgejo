// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgejo_migrations //nolint:revive

import "xorm.io/xorm"

func AddNormalizedFederatedURIToUser(x *xorm.Engine) error {
	type User struct {
		ID                     int64 `xorm:"pk autoincr"`
		NormalizedFederatedURI string
	}
	return x.Sync(&User{})
}
