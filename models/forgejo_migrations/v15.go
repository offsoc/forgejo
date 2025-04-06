// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgejo_migrations //nolint:revive

import (
	"time"

	"forgejo.org/modules/timeutil"

	"xorm.io/xorm"
)

type (
	SoftwareNameType string
)

type NodeInfo struct {
	SoftwareName SoftwareNameType
}

type FederationHost struct {
	ID             int64              `xorm:"pk autoincr"`
	HostFqdn       string             `xorm:"host_fqdn UNIQUE INDEX VARCHAR(255) NOT NULL"`
	NodeInfo       NodeInfo           `xorm:"extends NOT NULL"`
	LatestActivity time.Time          `xorm:"NOT NULL"`
	Created        timeutil.TimeStamp `xorm:"created"`
	Updated        timeutil.TimeStamp `xorm:"updated"`
}

func CreateFederationHostTable(x *xorm.Engine) error {
	return x.Sync(new(FederationHost))
}
