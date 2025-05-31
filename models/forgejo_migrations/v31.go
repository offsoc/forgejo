// Copyright 2025 The Forgejo Authors.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations //nolint:revive

import (
	"xorm.io/xorm"
	"xorm.io/xorm/schemas"
)

func SetTopicsAsEmptySlice(x *xorm.Engine) error {
	var err error
	if x.Dialect().URI().DBType == schemas.POSTGRES {
		_, err = x.Exec("UPDATE `repository` SET topics = '[]' WHERE topics IS NULL OR topics::text = 'null'")
	} else {
		_, err = x.Exec("UPDATE `repository` SET topics = '[]' WHERE topics IS NULL")
	}

	if err != nil {
		return err
	}

	type Repository struct {
		ID     int64    `xorm:"pk autoincr"`
		Topics []string `xorm:"TEXT JSON NOT NULL"`
	}

	return x.Sync(new(Repository))
}
