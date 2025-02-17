// Copyright 2025 The Forgejo Authors.
// SPDX-License-Identifier: GPL-3.0-or-later

package limiter

import (
	"cmp"
	"fmt"
	"net/netip"
	"slices"

	"code.forgejo.org/forgejo/ipranges"
)

type limiter struct {
	ipranges ipranges.IPRanges

	blocked []netip.Prefix
	allowed []netip.Prefix

	currentIP int
	ips       []netip.Addr
}

func New() Limiter {
	return new(limiter)
}

func (o *limiter) Init() error {
	o.ipranges = ipranges.New()
	return o.ipranges.Load()
}

func (o *limiter) SetMaxIPs(max int) {
	o.currentIP = 0
	o.ips = make([]netip.Addr, max)
}

func (o *limiter) GetMaxIPs() int {
	return cap(o.ips)
}

func convertStrings(cidrs []string) ([]netip.Prefix, error) {
	l := make([]netip.Prefix, 0, len(cidrs))
	for _, cidr := range cidrs {
		b, err := netip.ParsePrefix(cidr)
		if err != nil {
			return nil, err
		}
		l = append(l, b)
	}
	return l, nil
}

func convertCidrs(cidrs []netip.Prefix) []string {
	l := make([]string, 0, len(cidrs))
	for _, cidr := range cidrs {
		l = append(l, cidr.String())
	}
	return l
}

func (o *limiter) SetBlockList(cidrs []string) error {
	l, err := convertStrings(cidrs)
	if err != nil {
		return err
	}
	o.blocked = l
	return nil
}

func (o *limiter) GetBlockList() []string {
	return convertCidrs(o.blocked)
}

func (o *limiter) SetAllowList(cidrs []string) error {
	l, err := convertStrings(cidrs)
	if err != nil {
		return err
	}
	o.allowed = l
	return nil
}

func (o *limiter) GetAllowList() []string {
	return convertCidrs(o.allowed)
}

func (o *limiter) add(ip string) (netip.Addr, error) {
	a, err := netip.ParseAddr(ip)
	if err != nil {
		return netip.Addr{}, err
	}
	o.ips[o.currentIP] = a
	o.currentIP++
	if o.currentIP >= o.GetMaxIPs() {
		o.currentIP = 0
	}
	return a, nil
}

func find(cidrs []netip.Prefix, ip netip.Addr) (int, bool) {
	target := netip.PrefixFrom(ip, ip.BitLen())
	return slices.BinarySearchFunc(cidrs, target, func(e, t netip.Prefix) int {
		if e.Contains(ip) {
			return 0
		}
		if e.Addr().Less(ip) {
			return -1
		}
		return 1
	})
}

func (o *limiter) allow(ip netip.Addr) (allow bool, reason string, err error) {
	if o.blocked == nil {
		return true, "", nil
	}
	if _, found := find(o.allowed, ip); found {
		return true, "", nil
	}

	if n, found := find(o.blocked, ip); found {
		return false, fmt.Sprintf("%v %d", o.blocked[n], n), nil
	}

	return true, "", nil
}

func (o *limiter) AddAndAllow(ip string) (allow bool, reason string, err error) {
	a, err := o.add(ip)
	if err != nil {
		return false, "parse error", err
	}
	return o.allow(a)
}

func (o *limiter) MostUsedCidrs(top int) ([]CidrCount, []string) {
	ips := o.ips
	// ips - sort
	slices.SortFunc(ips, func(a, b netip.Addr) int {
		if a == b {
			return 0
		}
		if a.Less(b) {
			return -1
		}
		return 1
	})

	// ips - unique
	ipsLength := 1
	for i := 1; i < len(ips); i++ {
		if ips[i] == ips[i-1] {
			continue
		}
		ips[ipsLength] = ips[i]
		ipsLength++
	}

	cidrs := o.ipranges.Get()
	counts := make([]CidrCount, 0, len(cidrs))
	unknown := make([]string, 0, 100)
	ipIndex := 0
	for cidrIndex := 0; cidrIndex < len(cidrs) && ipIndex < ipsLength; cidrIndex++ {
		cidr := cidrs[cidrIndex]
		count := 0
		for ; ipIndex < ipsLength; ipIndex++ {
			ip := ips[ipIndex]
			if cidr.Contains(ip) {
				count++
			} else if ip.Less(cidr.Addr()) {
				unknown = append(unknown, ip.String())
			} else {
				break
			}
		}
		counts = append(counts, CidrCount{
			Cidr:  cidr.String(),
			Count: count,
		})
	}

	for _, ip := range ips[ipIndex:ipsLength] {
		unknown = append(unknown, ip.String())
	}

	slices.SortFunc(counts, func(a, b CidrCount) int {
		return cmp.Compare(b.Count, a.Count)
	})

	return counts[0:min(top, len(counts))], unknown
}
