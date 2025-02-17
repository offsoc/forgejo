// Copyright 2025 The Forgejo Authors.
// SPDX-License-Identifier: GPL-3.0-or-later

package limiter

type CidrCount struct {
	Cidr  string
	Count int
}

type Limiter interface {
	Init() error

	SetMaxIPs(max int)
	GetMaxIPs() int

	SetBlockList(cidrs []string) error
	GetBlockList() []string

	SetAllowList(cidrs []string) error
	GetAllowList() []string

	AddAndAllow(ip string) (allow bool, reason string, err error)

	MostUsedCidrs(top int) (cidrs []CidrCount, uknown []string)
}
