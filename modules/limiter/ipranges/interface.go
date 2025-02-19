// Copyright 2025 The Forgejo Authors.
// SPDX-License-Identifier: GPL-3.0-or-later

package ipranges

import (
	"time"
)

type Log interface{}

type StatsSample interface {
	UniqueIPs() int
	BlockedIPs() int
}

type Stats interface {
	Start() time.Time
	Duration() time.Duration
	Samples() []StatsSample
}

type (
	ErrorNotExcessive     struct{ error }
	ErrorTargetNotReached struct{ error }
)

type IPRanges interface {
	Init() error

	SetMaxIPs(max int)
	GetMaxIPs() int

	SetBlockList(cidrs []string) error
	GetBlockList() []string

	SetAllowList(cidrs []string) error
	GetAllowList() []string

	AddAndAllow(ip string) (allow bool, reason string, err error)

	GetLog() Log
	ResetLog()

	Simulation(log Log, blocked, allowed []string) error
	GetStats(log Log, start time.Time, duration time.Duration) Stats
	CompileBlockList(log Log, target, excessive int) (blockList, unknown []string, err error)
}
