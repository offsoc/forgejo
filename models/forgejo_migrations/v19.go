// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgejo_migrations //nolint:revive

import "xorm.io/xorm"

func AddExternalURLColumnToAttachmentTable(x *xorm.Engine) error {
	type Attachment struct {
		ID          int64 `xorm:"pk autoincr"`
		ExternalURL string
	}
	return x.Sync(new(Attachment))
}

