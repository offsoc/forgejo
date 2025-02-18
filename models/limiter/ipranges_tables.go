// Copyright 2025 The Forgejo Authors.
// SPDX-License-Identifier: GPL-3.0-or-later

package limiter

import (
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/modules/timeutil"
)

func init() {
	db.RegisterModel(new(IPRangesSettings))
	db.RegisterModel(new(IPRangeList))
}

type IPRangesSettings struct {
	ID      int64 `xorm:"pk autoincr"`
	Enabled bool

	ExpectedIPCount  int
	ExcessiveIPCount int
	BlockTop         int
	Periodicity      string

	UpdatedUnix timeutil.TimeStamp `xorm:"updated index"`
}

func (o IPRangesSettings) TableName() string {
	return "limiter_ipranges_settings"
}

type IPRangePurpose string

const (
	IPRangeBlock IPRangePurpose = "block"
	IPRangeAllow IPRangePurpose = "allow"
)

type IPRangeList struct {
	ID      int64 `xorm:"pk autoincr"`
	Purpose IPRangePurpose
	Cidr    string
}

func (o IPRangeList) TableName() string {
	return "limiter_iprange_list"
}
