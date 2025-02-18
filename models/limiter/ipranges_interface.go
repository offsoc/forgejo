// Copyright 2025 The Forgejo Authors.
// SPDX-License-Identifier: GPL-3.0-or-later

package limiter

import (
	"context"
)

type IPRanges interface {
	SetEnabled(ctx context.Context, enabled bool) error
	GetEnabled(ctx context.Context) bool

	SetExpectedIPCount(ctx context.Context, value int) error
	GetExpectedIPCount(ctx context.Context) int

	SetExcessiveIPCount(ctx context.Context, value int) error
	GetExcessiveIPCount(ctx context.Context) int

	SetBlockTop(ctx context.Context, value int) error
	GetBlockTop(ctx context.Context) int

	SetPeriodicity(ctx context.Context, value string) error
	GetPeriodicity(ctx context.Context) string

	SetBlocked(ctx context.Context, ipranges []string) error
	GetBlocked(ctx context.Context) ([]string, error)

	SetAllowed(ctx context.Context, ipranges []string) error
	GetAllowed(ctx context.Context) ([]string, error)
}
