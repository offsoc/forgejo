// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgejo_migrations //nolint:revive

import (
	"context"
	"crypto/md5"
	"encoding/base64"

	"code.gitea.io/gitea/models/auth"
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/modules/secret"
	"code.gitea.io/gitea/modules/setting"

	"xorm.io/xorm"
	"xorm.io/xorm/schemas"
)

func MigrateTwoFactorToKeying(x *xorm.Engine) error {
	var err error

	switch x.Dialect().URI().DBType {
	case schemas.MYSQL:
		_, err = x.Exec("ALTER TABLE `two_factor` MODIFY `secret` BLOB")
	case schemas.POSTGRES:
		_, err = x.Exec("ALTER TABLE `two_factor` ALTER COLUMN `secret` SET DATA TYPE bytea USING secret::text::bytea")
	}
	if err != nil {
		return err
	}

	oldEncryptionKey := md5.Sum([]byte(setting.SecretKey))

	return db.Iterate(context.Background(), nil, func(ctx context.Context, bean *auth.TwoFactor) error {
		decodedStoredSecret, err := base64.StdEncoding.DecodeString(string(bean.Secret))
		if err != nil {
			return err
		}

		secretBytes, err := secret.AesDecrypt(oldEncryptionKey[:], decodedStoredSecret)
		if err != nil {
			return err
		}

		bean.SetSecret(string(secretBytes))
		_, err = db.GetEngine(ctx).Cols("secret").ID(bean.ID).Update(bean)
		return err
	})
}
