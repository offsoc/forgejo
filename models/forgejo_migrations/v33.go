// Copyright 2025 The Forgejo Authors. All rights reserved.
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
		UserID       int64 `xorm:"NOT NULL INDEX user_id"`
		ActorID      int64
		ActorURI     string
		NoteContent  string             `xorm:"TEXT"`
		NoteURL      string             `xorm:"VARCHAR(255)"`
		OriginalNote string             `xorm:"TEXT"`
		Created      timeutil.TimeStamp `xorm:"created"`
	}

	// drop unique index on HostFqdn & add unique index on HostFqdn+HostPort
	type FederationHost struct {
		ID       int64  `xorm:"pk autoincr"`
		HostFqdn string `xorm:"host_fqdn UNIQUE(federation_host) INDEX VARCHAR(255) NOT NULL"`
		HostPort uint16 `xorm:"UNIQUE(federation_host) INDEX NOT NULL DEFAULT 443"`
	}

	type FederatedUserFollower struct {
		ID int64 `xorm:"pk autoincr"`

		FollowedUserID  int64 `xorm:"NOT NULL unique(fuf_rel)"`
		FollowingUserID int64 `xorm:"NOT NULL unique(fuf_rel)"`
	}

	// Add InboxPath to FederatedUser & add index fo UserID
	type FederatedUser struct {
		ID        int64 `xorm:"pk autoincr"`
		UserID    int64 `xorm:"NOT NULL INDEX user_id"`
		InboxPath string
	}

	var err error

	federationHostTable, err := x.TableInfo(FederationHost{})
	if err != nil {
		log.Warn("migration[33]: There was an issue: %v", err)
		return nil
	}
	for _, index := range federationHostTable.Indexes {
		if index.Name == "host_fqdn" {
			sessMigration := x.NewSession()
			defer sessMigration.Close()

			if err := sessMigration.Begin(); err != nil {
				return nil
			}
			sql := x.Dialect().DropIndexSQL(federationHostTable.Name, index)
			_, err := sessMigration.Exec(sql)
			if err != nil {
				log.Warn("migration[33]: There was an issue: %v", err)
			}
			err = sessMigration.Commit()
			if err != nil {
				log.Warn("migration[33]: There was an issue: %v", err)
			}
		}
	}

	err = x.Sync(&FederationHost{})
	if err != nil {
		log.Warn("migration[33]: There was an issue: %v", err)
		return nil
	}

	err = x.Sync(&FederatedUserActivity{})
	if err != nil {
		log.Warn("migration[33]: There was an issue: %v", err)
		return nil
	}

	err = x.Sync(&FederatedUserFollower{})
	if err != nil {
		log.Warn("migration[33]: There was an issue: %v", err)
		return nil
	}

	err = x.Sync(&FederatedUser{})
	if err != nil {
		log.Warn("migration[33]: There was an issue: %v", err)
		return nil
	}

	// Migrate
	sessMigration := x.NewSession()
	defer sessMigration.Close()
	if err := sessMigration.Begin(); err != nil {
		log.Warn("migration[33]: There was an issue: %v", err)
		return nil
	}
	federatedUsers := make([]*FederatedUser, 0)
	err = sessMigration.OrderBy("id").Find(&federatedUsers)
	if err != nil {
		log.Warn("migration[33]: There was an issue: %v", err)
		return nil
	}

	for _, federatedUser := range federatedUsers {
		if federatedUser.InboxPath != "" {
			log.Info("migration[33]: There was an issue: %v", federatedUser)
		} else {
			// Migrate User.InboxPath
			sql := "UPDATE `federated_user` SET `inbox_path` = ? WHERE `id` = ?"
			if _, err := sessMigration.Exec(sql, fmt.Sprintf("/api/v1/activitypub/user-id/%v/inbox", federatedUser.UserID), federatedUser.ID); err != nil {
				log.Warn("migration[33]: There was an issue: %v", err)
				return nil
			}
		}
	}

	err = sessMigration.Commit()
	if err != nil {
		log.Warn("migration[33]: There was an issue: %v", err)
		return nil
	}
	return nil
}
