// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgejo_migrations //nolint:revive

import "xorm.io/xorm"

func AddLegacyToWebAuthnCredential(x *xorm.Engine) error {
	type WebauthnCredential struct {
		ID             int64 `xorm:"pk autoincr"`
		BackupEligible bool  `xorm:"NOT NULL DEFAULT false"`
		BackupState    bool  `xorm:"NOT NULL DEFAULT false"`
		Legacy         bool  `xorm:"NOT NULL DEFAULT true"`
	}

	return x.Sync(&WebauthnCredential{})
}
