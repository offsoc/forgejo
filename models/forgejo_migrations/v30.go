// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgejo_migrations //nolint:revive

import (
	"time"

	"forgejo.org/models/migrations/base"
	"forgejo.org/modules/forgefed"
	"forgejo.org/modules/log"
	"forgejo.org/modules/timeutil"

	"xorm.io/xorm"
)

func MigrateNormalizedFederatedURI(x *xorm.Engine) error {
	// Update schema
	type FederatedUser struct {
		ID                    int64  `xorm:"pk autoincr"`
		UserID                int64  `xorm:"NOT NULL"`
		ExternalID            string `xorm:"UNIQUE(federation_user_mapping) NOT NULL"`
		FederationHostID      int64  `xorm:"UNIQUE(federation_user_mapping) NOT NULL"`
		NormalizedOriginalURL string
	}
	type User struct {
		ID                     int64 `xorm:"pk autoincr"`
		NormalizedFederatedURI string
	}
	type FederationHost struct {
		ID             int64              `xorm:"pk autoincr"`
		HostFqdn       string             `xorm:"host_fqdn UNIQUE INDEX VARCHAR(255) NOT NULL"`
		NodeInfo       NodeInfo           `xorm:"extends NOT NULL"`
		HostPort       uint16             `xorm:"NOT NULL DEFAULT 443"`
		HostSchema     string             `xorm:"NOT NULL DEFAULT 'https'"`
		LatestActivity time.Time          `xorm:"NOT NULL"`
		Created        timeutil.TimeStamp `xorm:"created"`
		Updated        timeutil.TimeStamp `xorm:"updated"`
	}
	if err := x.Sync(new(User), new(FederatedUser), new(FederationHost)); err != nil {
		return err
	}

	// Migrate
	sessMigration := x.NewSession()
	defer sessMigration.Close()
	if err := sessMigration.Begin(); err != nil {
		return err
	}
	federatedUsers := make([]*FederatedUser, 0)
	err := sessMigration.OrderBy("id").Find(&federatedUsers)
	if err != nil {
		return err
	}

	for _, federatedUser := range federatedUsers {
		if federatedUser.NormalizedOriginalURL != "" {
			log.Trace("migration[30]: FederatedUser was already migrated %v", federatedUser)
		} else {
			user := &User{}
			has, err := sessMigration.Where("id=?", federatedUser.UserID).Get(user)
			if err != nil {
				return err
			}

			if !has {
				log.Debug("migration[30]: User missing for federated user: %v", federatedUser)
				_, err := sessMigration.Delete(federatedUser)
				if err != nil {
					return err
				}
			} else {
				// Migrate User.NormalizedFederatedURI -> FederatedUser.NormalizedOriginalUrl
				sql := "UPDATE `federated_user` SET `normalized_original_url` = ? WHERE `id` = ?"
				if _, err := sessMigration.Exec(sql, user.NormalizedFederatedURI, federatedUser.FederationHostID); err != nil {
					return err
				}

				// Migrate (Port, Schema) FederatedUser.NormalizedOriginalUrl -> FederationHost.(Port, Schema)
				actorID, err := forgefed.NewActorID(user.NormalizedFederatedURI)
				if err != nil {
					return err
				}
				sql = "UPDATE `federation_host` SET `host_port` = ?, `host_schema` = ? WHERE `id` = ?"
				if _, err := sessMigration.Exec(sql, actorID.HostPort, actorID.HostSchema, federatedUser.FederationHostID); err != nil {
					return err
				}
			}
		}
	}

	if err := sessMigration.Commit(); err != nil {
		return err
	}

	// Drop User.NormalizedFederatedURI field in extra transaction
	sessSchema := x.NewSession()
	defer sessSchema.Close()
	if err := sessSchema.Begin(); err != nil {
		return err
	}
	if err := base.DropTableColumns(sessSchema, "user", "normalized_federated_uri"); err != nil {
		return err
	}
	return sessSchema.Commit()
}
