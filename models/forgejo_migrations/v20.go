// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgejo_migrations //nolint:revive

import "xorm.io/xorm"

type (
	QuotaLimitSubject  int
	QuotaLimitSubjects []QuotaLimitSubject

	QuotaKind int
)

type QuotaRule struct {
	Name     string `xorm:"pk not null"`
	Limit    int64  `xorm:"NOT NULL"`
	Subjects QuotaLimitSubjects
}

type QuotaGroup struct {
	Name string `xorm:"pk NOT NULL"`
}

type QuotaGroupRuleMapping struct {
	ID        int64  `xorm:"pk autoincr"`
	GroupName string `xorm:"index unique(qgrm_gr) not null"`
	RuleName  string `xorm:"unique(qgrm_gr) not null"`
}

type QuotaGroupMapping struct {
	ID        int64     `xorm:"pk autoincr"`
	Kind      QuotaKind `xorm:"unique(qgm_kmg) not null"`
	MappedID  int64     `xorm:"unique(qgm_kmg) not null"`
	GroupName string    `xorm:"index unique(qgm_kmg) not null"`
}

func CreateQuotaTables(x *xorm.Engine) error {
	if err := x.Sync(new(QuotaRule)); err != nil {
		return err
	}

	if err := x.Sync(new(QuotaGroup)); err != nil {
		return err
	}

	if err := x.Sync(new(QuotaGroupRuleMapping)); err != nil {
		return err
	}

	return x.Sync(new(QuotaGroupMapping))
}
