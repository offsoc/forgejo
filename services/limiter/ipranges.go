// Copyright 2025 The Forgejo Authors.
// SPDX-License-Identifier: GPL-3.0-or-later

package limiter

import (
	"context"

	limiter_model "code.gitea.io/gitea/models/limiter"
	limiter_module "code.gitea.io/gitea/modules/limiter"
)

type ipranges struct {
	model  limiter_model.IPRanges
	module limiter_module.IPRanges
}

func (o *ipranges) Cron(ctx context.Context) error {
	blocked, err := o.GetModel().GetBlocked(ctx)
	if err != nil {
		return err
	}
	if err := o.GetModule().SetBlockList(blocked); err != nil {
		return err
	}

	allowed, err := o.GetModel().GetAllowed(ctx)
	if err != nil {
		return err
	}
	if err := o.GetModule().SetAllowList(allowed); err != nil {
		return err
	}

	cidrCount, unknown := o.GetModule().MostUsedCidrs(o.GetModel().GetBlockTop(ctx))

	return nil
}

func (o *ipranges) GetModel() limiter_model.IPRanges {
	return o.model
}

func (o *ipranges) GetModule() limiter_module.IPRanges {
	return o.module
}

const maxIPs = 10000000

func (o *ipranges) Init(ctx context.Context) {
	o.module.SetMaxIPs(maxIPs)
}

var singleton IPRanges

func IPRangesSingleton() IPRanges {
	return singleton
}

func NewIPRanges(ctx context.Context) (IPRanges, error) {
	o := new(ipranges)

	model, err := limiter_model.NewIPRanges(ctx)
	if err != nil {
		return nil, err
	}

	module := limiter_module.NewIPRanges()
	if err := o.module.Init(); err != nil {
		return nil, err
	}

	o.model = model
	o.module = module

	return o, nil
}

func Init(ctx context.Context) error {
	var err error
	singleton, err = NewIPRanges(ctx)
	return err
}
