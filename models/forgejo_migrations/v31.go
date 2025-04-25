// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgejo_migrations //nolint:revive

import (
	"fmt"

	"forgejo.org/modules/log"
	"forgejo.org/modules/timeutil"

	"xorm.io/xorm"
)

func AddFederatedUserActivityTables(x *xorm.Engine) error {
	type FederatedUserActivity struct {
		ID           int64 `xorm:"pk autoincr"`
		UserID       int64 `xorm:"NOT NULL"`
		ActorID      string
		NoteContent  string
		NoteURL      string
		OriginalNote string
		Created      timeutil.TimeStamp `xorm:"created"`
	}

	type FederatedUserFollower struct {
		ID int64 `xorm:"pk autoincr"`

		FollowedUserID  int64 `xorm:"NOT NULL unique(fuf_rel)"`
		FollowingUserID int64 `xorm:"NOT NULL unique(fuf_rel)"`
	}

	// Add InboxPath to FederatedUser
	type FederatedUser struct {
		ID        int64 `xorm:"pk autoincr"`
		UserID    int64 `xorm:"NOT NULL"`
		InboxPath string
		ActorURL  *string
	}

	err := x.Sync(&FederatedUserActivity{})
	if err != nil {
		return err
	}

	err = x.Sync(&FederatedUserFollower{})
	if err != nil {
		return err
	}

	err = x.Sync(&FederatedUser{})
	if err != nil {
		return err
	}

	// Migrate
	sessMigration := x.NewSession()
	defer sessMigration.Close()
	if err := sessMigration.Begin(); err != nil {
		return err
	}
	federatedUsers := make([]*FederatedUser, 0)
	err = sessMigration.OrderBy("id").Find(&federatedUsers)
	if err != nil {
		return err
	}

	for _, federatedUser := range federatedUsers {
		if federatedUser.InboxPath != "" {
			log.Trace("migration[31]: FederatedUser was already migrated %v", federatedUser)
		} else {
			// Migrate User.InboxPath
			sql := "UPDATE `federated_user` SET `inbox_path` = ? WHERE `id` = ?"
			if _, err := sessMigration.Exec(sql, fmt.Sprintf("/api/v1/activitypub/user-id/%v/inbox", federatedUser.UserID), federatedUser.ID); err != nil {
				return err
			}
		}
	}

	return sessMigration.Commit()
}
