// Copyright 2025 The Forgejo Authors.
// SPDX-License-Identifier: GPL-3.0-or-later

package limiter

import (
	"context"

	limiter_model "code.gitea.io/gitea/models/limiter"
	limiter_module "code.gitea.io/gitea/modules/limiter"
)

type IPRanges interface {
	Init(ctx context.Context)

	GetModel() limiter_model.IPRanges
	GetModule() limiter_module.IPRanges

	Cron(ctx context.Context) error
}
