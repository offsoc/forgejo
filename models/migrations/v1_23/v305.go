// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package v1_23 //nolint

import (
	"code.gitea.io/gitea/models/migrations/base"
	"xorm.io/xorm"
)

func MigrateNormalizedFederatedURI(x *xorm.Engine) error {
	// New Fields
	type FederatedUser struct {
		ID                    int64  `xorm:"pk autoincr"`
		UserID                int64  `xorm:"NOT NULL"`
		ExternalID            string `xorm:"UNIQUE(federation_user_mapping) NOT NULL"`
		FederationHostID      int64  `xorm:"UNIQUE(federation_user_mapping) NOT NULL"`
		NormalizedOriginalURL string
	}
	type User struct {
		NormalizedFederatedURI string
	}
	// TODO: add new fields to FederationHost
	if err := x.Sync(new(User), new(FederatedUser)); err != nil {
		return err
	}

	// Migrate User.NormalizedFederatedURI -> FederatedUser.NormalizedOriginalUrl
	sessMigration := x.NewSession()
	defer sessMigration.Close()
	if err := sessMigration.Begin(); err != nil {
		return err
	}
	if _, err := sessMigration.Exec("UPDATE `federated_user` SET `normalized_original_url` = (SELECT normalized_federated_uri FROM `user` WHERE `user`.id = federated_user.user_id)"); err != nil {
		return err
	}
	if err := sessMigration.Commit(); err != nil {
		return err
	}

	// Migrate (Port, Schema) FederatedUser.NormalizedOriginalUrl -> FederationHost.(Port, Schema)
	// TODO

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
