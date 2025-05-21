// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgejo_migrations //nolint:revive

import (
	"xorm.io/xorm"
	"xorm.io/xorm/schemas"
)

func SetTopicsAsEmptySlice(x *xorm.Engine) error {
	if x.Dialect().URI().DBType == schemas.POSTGRES {
		_, err := x.Exec("UPDATE `repository` SET topics = '[]' WHERE topics IS NULL OR topics::text = 'null'")
		return err
	}

	_, err := x.Exec("UPDATE `repository` SET topics = '[]' WHERE topics IS NULL")
	return err
}
