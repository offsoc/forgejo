// Copyright 2025 The Forgejo Authors.
// SPDX-License-Identifier: GPL-3.0-or-later

package ipranges

import (
	"time"
)

type statsSample struct {
	blocked int
	total   int
}

func (o statsSample) BlockedIPs() int { return o.blocked }
func (o statsSample) UniqueIPs() int  { return o.total }

type stats struct {
	start    time.Time
	duration time.Duration
	samples  []StatsSample
}

func (o stats) Start() time.Time        { return o.start }
func (o stats) Duration() time.Duration { return o.duration }
func (o stats) Samples() []StatsSample  { return o.samples }
