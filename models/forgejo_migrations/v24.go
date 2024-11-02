// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgejo_migrations //nolint:revive

import (
	"code.gitea.io/gitea/modules/timeutil"

	"xorm.io/xorm"
)

func AddFederatedUserActivityTables(x *xorm.Engine) error {
	type FederatedUserActivity struct {
		ID     int64 `xorm:"pk autoincr"`
		UserID int64 `xorm:"NOT NULL"`

		ExternalID  string `xorm:"NOT NULL"`
		Note        string
		OriginalURL string

		Original string

		Created timeutil.TimeStamp `xorm:"created"`
	}

	type FederatedUserFollower struct {
		ID int64 `xorm:"pk autoincr"`

		LocalUserID     int64 `xorm:"NOT NULL unique(fuf_rel)"`
		FederatedUserID int64 `xorm:"NOT NULL unique(fuf_rel)"`
	}

	// Add ActorURL and InboxURL to FederatedUser
	type FederatedUser struct {
		ID int64 `xorm:"pk autoincr"`

		InboxURL *string
		ActorURL *string
	}

	err := x.Sync(&FederatedUserActivity{})
	if err != nil {
		return err
	}

	err = x.Sync(&FederatedUserFollower{})
	if err != nil {
		return err
	}

	return x.Sync(&FederatedUser{})
}
