// Copyright 2023 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgejo_v1_20 //nolint:revive

import (
	"forgejo.org/modules/timeutil"

	"xorm.io/xorm"
)

type AuthorizationToken struct {
	ID              int64  `xorm:"pk autoincr"`
	UID             int64  `xorm:"INDEX"`
	LookupKey       string `xorm:"INDEX UNIQUE"`
	HashedValidator string
	Expiry          timeutil.TimeStamp
}

func (AuthorizationToken) TableName() string {
	return "forgejo_auth_token"
}

func CreateAuthorizationTokenTable(x *xorm.Engine) error {
	return x.Sync(new(AuthorizationToken))
}
