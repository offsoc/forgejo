// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user

import (
	
)

type FederatedUserFollower struct {
	ID int64 `xorm:"pk autoincr"`

	LocalUserID     int64 `xorm:"NOT NULL unique(fuf_rel)"`
	FederatedUserID int64 `xorm:"NOT NULL unique(fuf_rel)"`
}
