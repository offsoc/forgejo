// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgejo_migrations //nolint:revive

import "xorm.io/xorm"

func AddHashBlake2bToPackageBlob(x *xorm.Engine) error {
	type PackageBlob struct {
		ID          int64  `xorm:"pk autoincr"`
		HashBlake2b string `xorm:"hash_blake2b char(128) UNIQUE(blake2b) INDEX"`
	}
	return x.Sync(&PackageBlob{})
}
