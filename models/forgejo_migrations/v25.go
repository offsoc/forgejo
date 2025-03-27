// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgejo_migrations //nolint:revive

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"fmt"

	"forgejo.org/models/auth"
	"forgejo.org/models/db"
	"forgejo.org/modules/log"
	"forgejo.org/modules/secret"
	"forgejo.org/modules/setting"

	"xorm.io/xorm"
	"xorm.io/xorm/schemas"
)

func MigrateTwoFactorToKeying(x *xorm.Engine) error {
	var err error

	// When upgrading from Forgejo v9 to v10, this migration will already be
	// called from models/migrations/migrations.go migration 304 and must not
	// be run twice.
	var version int
	_, err = x.Table("version").Where("`id` = 1").Select("version").Get(&version)
	if err != nil {
		// the version table does not exist when a test environment only applies Forgejo migrations
	} else if version > 304 {
		return nil
	}

	switch x.Dialect().URI().DBType {
	case schemas.MYSQL:
		_, err = x.Exec("ALTER TABLE `two_factor` MODIFY `secret` BLOB")
	case schemas.SQLITE:
		_, err = x.Exec("ALTER TABLE `two_factor` RENAME COLUMN `secret` TO `secret_backup`")
		if err != nil {
			return err
		}
		_, err = x.Exec("ALTER TABLE `two_factor` ADD COLUMN `secret` BLOB")
		if err != nil {
			return err
		}
		_, err = x.Exec("UPDATE `two_factor` SET `secret` = `secret_backup`")
		if err != nil {
			return err
		}
		_, err = x.Exec("ALTER TABLE `two_factor` DROP COLUMN `secret_backup`")
	case schemas.POSTGRES:
		_, err = x.Exec("ALTER TABLE `two_factor` ALTER COLUMN `secret` SET DATA TYPE bytea USING secret::text::bytea")
	}
	if err != nil {
		return err
	}

	oldEncryptionKey := md5.Sum([]byte(setting.SecretKey))

	messages := make([]string, 0, 100)
	ids := make([]int64, 0, 100)

	err = db.Iterate(context.Background(), nil, func(ctx context.Context, bean *auth.TwoFactor) error {
		decodedStoredSecret, err := base64.StdEncoding.DecodeString(string(bean.Secret))
		if err != nil {
			messages = append(messages, fmt.Sprintf("two_factor.id=%d, two_factor.uid=%d: base64.StdEncoding.DecodeString: %v", bean.ID, bean.UID, err))
			ids = append(ids, bean.ID)
			return nil
		}

		secretBytes, err := secret.AesDecrypt(oldEncryptionKey[:], decodedStoredSecret)
		if err != nil {
			messages = append(messages, fmt.Sprintf("two_factor.id=%d, two_factor.uid=%d: secret.AesDecrypt: %v", bean.ID, bean.UID, err))
			ids = append(ids, bean.ID)
			return nil
		}

		bean.SetSecret(string(secretBytes))
		_, err = db.GetEngine(ctx).Cols("secret").ID(bean.ID).Update(bean)
		return err
	})
	if err == nil {
		if len(ids) > 0 {
			log.Error("Forgejo migration[25]: The following TOTP secrets were found to be corrupted and removed from the database. TOTP is no longer required to login with the associated users. They should be informed because they will need to visit their security settings and configure TOTP again. No other action is required. See https://codeberg.org/forgejo/forgejo/issues/6637 for more context on the various causes for such a corruption.")
			for _, message := range messages {
				log.Error("Forgejo migration[25]: %s", message)
			}

			_, err = db.GetEngine(context.Background()).In("id", ids).NoAutoCondition().NoAutoTime().Delete(&auth.TwoFactor{})
		}
	}

	return err
}
