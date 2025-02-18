// Copyright 2025 The Forgejo Authors.
// SPDX-License-Identifier: GPL-3.0-or-later

package limiter

import (
	"context"

	"code.gitea.io/gitea/models/db"
	"xorm.io/builder"
)

func NewIPRanges(ctx context.Context) (IPRanges, error) {
	o := new(ipRanges)

	has, err := db.GetEngine(ctx).Get(&o.settings)
	if err != nil {
		return nil, err
	}
	if !has {
		if _, err := db.GetEngine(ctx).Insert(&o.settings); err != nil {
			return nil, err
		}
	}

	return o, nil
}

type ipRanges struct {
	settings IPRangesSettings
}

func (o *ipRanges) updateSettings(ctx context.Context) error {
	_, err := db.GetEngine(ctx).ID(o.settings.ID).Update(&o.settings)
	return err
}

func (o *ipRanges) SetEnabled(ctx context.Context, enabled bool) error {
	o.settings.Enabled = enabled
	return o.updateSettings(ctx)
}

func (o *ipRanges) GetEnabled(ctx context.Context) bool {
	return o.settings.Enabled
}

func (o *ipRanges) SetExpectedIPCount(ctx context.Context, value int) error {
	o.settings.ExpectedIPCount = value
	return o.updateSettings(ctx)
}

func (o *ipRanges) GetExpectedIPCount(ctx context.Context) int {
	return o.settings.ExpectedIPCount
}

func (o *ipRanges) SetExcessiveIPCount(ctx context.Context, value int) error {
	o.settings.ExcessiveIPCount = value
	return o.updateSettings(ctx)
}

func (o *ipRanges) GetExcessiveIPCount(ctx context.Context) int {
	return o.settings.ExcessiveIPCount
}

func (o *ipRanges) SetBlockTop(ctx context.Context, value int) error {
	o.settings.BlockTop = value
	return o.updateSettings(ctx)
}

func (o *ipRanges) GetBlockTop(ctx context.Context) int {
	return o.settings.BlockTop
}

func (o *ipRanges) SetPeriodicity(ctx context.Context, value string) error {
	o.settings.Periodicity = value
	return o.updateSettings(ctx)
}

func (o *ipRanges) GetPeriodicity(ctx context.Context) string {
	return o.settings.Periodicity
}

func (o *ipRanges) updateRangeList(ctx context.Context, purpose IPRangePurpose, ipranges []string) error {
	ctx, committer, err := db.TxContext(ctx)
	if err != nil {
		return err
	}
	defer committer.Close()

	ipRangeList := make([]IPRangeList, 0, len(ipranges))

	for _, iprange := range ipranges {
		ipRangeList = append(ipRangeList, IPRangeList{
			Purpose: purpose,
			Cidr:    iprange,
		})
	}

	sess := db.GetEngine(ctx)
	_, err = sess.Where("purpose = ?", purpose).Delete(new(IPRangeList))
	if err != nil {
		return err
	}
	_, err = sess.Insert(ipRangeList)
	if err != nil {
		return err
	}

	return committer.Commit()
}

func (o *ipRanges) getRangeList(ctx context.Context, purpose IPRangePurpose) ([]string, error) {
	ipranges := make([]string, 0, 100)
	if err := db.Iterate(ctx, builder.Eq{"purpose": purpose}, func(ctx context.Context, iprange *IPRangeList) error {
		ipranges = append(ipranges, iprange.Cidr)
		return nil
	}); err != nil {
		return nil, err
	}

	return ipranges, nil
}

func (o *ipRanges) SetBlocked(ctx context.Context, ipranges []string) error {
	return o.updateRangeList(ctx, IPRangeBlock, ipranges)
}

func (o *ipRanges) GetBlocked(ctx context.Context) ([]string, error) {
	return o.getRangeList(ctx, IPRangeBlock)
}

func (o *ipRanges) SetAllowed(ctx context.Context, ipranges []string) error {
	return o.updateRangeList(ctx, IPRangeBlock, ipranges)
}

func (o *ipRanges) GetAllowed(ctx context.Context) ([]string, error) {
	return o.getRangeList(ctx, IPRangeBlock)
}
