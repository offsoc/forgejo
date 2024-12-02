// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package git

import (
	"code.gitea.io/gitea/services/context"
)

type serviceHandlerBase interface {
	Init(ctx *context.Context) bool
	GetRepoPath() string
	GetEnviron() []string
}
