// Copyright 2025 The Forgejo Authors.
// SPDX-License-Identifier: GPL-3.0-or-later

package ipranges

import (
	"cmp"
	"fmt"
	"net/netip"
	"slices"
	"sync"
	"time"

	"code.gitea.io/gitea/util/wraparoundcontainer"

	"code.forgejo.org/forgejo/ipranges"
)

type cidrCount struct {
	cidr  netip.Prefix
	count int
}

type limiter struct {
	ipranges ipranges.IPRanges

	blockedOrAllowedMutex RWLocker
	blocked               []netip.Prefix
	allowed               []netip.Prefix

	log *log
}

func NewIPRanges() IPRanges {
	return new(limiter).initLocks()
}

func (o *limiter) initLocks() *limiter {
	o.blockedOrAllowedMutex = &sync.RWMutex{}
	return o
}

func (o *limiter) Init() error {
	o.ipranges = ipranges.New()
	o.SetMaxIPs(100)
	return o.ipranges.Load()
}

func (o *limiter) SetMaxIPs(max int) {
	o.log.init(max)
}

func (o *limiter) GetMaxIPs() int {
	return o.log.capacity()
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
	o.blockedOrAllowedMutex.Lock()
	o.blocked = l
	o.blockedOrAllowedMutex.Unlock()
	return nil
}

func (o *limiter) GetBlockList() []string {
	o.blockedOrAllowedMutex.RLock()
	defer o.blockedOrAllowedMutex.RUnlock()
	return convertCidrs(o.blocked)
}

func (o *limiter) SetAllowList(cidrs []string) error {
	l, err := convertStrings(cidrs)
	if err != nil {
		return err
	}
	o.blockedOrAllowedMutex.Lock()
	o.allowed = l
	o.blockedOrAllowedMutex.Unlock()
	return nil
}

func (o *limiter) GetAllowList() []string {
	o.blockedOrAllowedMutex.RLock()
	defer o.blockedOrAllowedMutex.RUnlock()
	return convertCidrs(o.allowed)
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

func allow(ip netip.Addr, allowed, blocked []netip.Prefix) (bool, string) {
	if blocked == nil {
		return true, ""
	}
	if _, found := find(allowed, ip); found {
		return true, ""
	}

	if n, found := find(blocked, ip); found {
		return false, fmt.Sprintf("%v %d", blocked[n], n)
	}

	return true, ""
}

func addAndAllow(log *log, ip netip.Addr, allowed, blocked []netip.Prefix) (bool, string) {
	ok, reason := allow(ip, allowed, blocked)
	log.add(ip, !ok)
	return ok, reason
}

func (o *limiter) AddAndAllow(ip string) (bool, string, error) {
	a, err := netip.ParseAddr(ip)
	if err != nil {
		return false, "parse error", err
	}

	o.blockedOrAllowedMutex.RLock()
	defer o.blockedOrAllowedMutex.RUnlock()

	allowed, reason := addAndAllow(o.log, a, o.allowed, o.blocked)
	return allowed, reason, nil
}

func (o *limiter) getCidrCount(log *log) ([]cidrCount, int, []string) {
	log.sortByIP()
	ips := uniqueIPs(log.all())
	cidrs := o.ipranges.Get()
	counts := make([]cidrCount, 0, len(cidrs))
	unknown := make([]string, 0, 100)
	ipIndex := 0
	for cidrIndex := 0; cidrIndex < len(cidrs) && ipIndex < len(ips); cidrIndex++ {
		cidr := cidrs[cidrIndex]
		count := 0
		for ; ipIndex < len(ips); ipIndex++ {
			ip := ips[ipIndex]
			if cidr.Contains(ip) {
				count++
			} else if ip.Less(cidr.Addr()) {
				unknown = append(unknown, ip.String())
			} else {
				break
			}
		}
		counts = append(counts, cidrCount{
			cidr:  cidr,
			count: count,
		})
	}

	for _, ip := range ips[ipIndex:] {
		unknown = append(unknown, ip.String())
	}

	slices.SortFunc(counts, func(a, b cidrCount) int {
		return cmp.Compare(b.count, a.count)
	})

	return counts, len(ips), unknown
}

func (o *limiter) getStat(entries []logEntry) statsSample {
	s := statsSample{}
	for _, entry := range entriesByUniqueIP(entries) {
		if entry.blocked {
			s.blocked++
		}
		s.total++
	}
	return s
}

func (o *limiter) GetStats(l Log, start time.Time, duration time.Duration) Stats {
	log := l.(*log)
	log.sortByTime()
	all := log.all()
	i, _ := slices.BinarySearchFunc(all, logEntry{created: start}, func(e, t logEntry) int {
		return cmp.Compare(e.created.Unix(), t.created.Unix())
	})
	samples := make([]StatsSample, 0, 100)
	entries := make([]logEntry, 0, 100)
	intervalStart := start
	for _, entry := range all[i:] {
		if entry.created.Sub(intervalStart) > duration {
			samples = append(samples, o.getStat(entries))
			entries = entries[:0]
			intervalStart.Add(duration)
		}
		entries = append(entries, entry)
	}

	if len(entries) > 0 {
		samples = append(samples, o.getStat(entries))
	}

	return stats{
		start:    start,
		duration: duration,
		samples:  samples,
	}
}

func simulation(log *log, blocked, allowed []netip.Prefix) {
	for _, entry := range log.all() {
		ok, _ := allow(entry.ip, allowed, blocked)
		entry.blocked = !ok
	}
}

func (o *limiter) Simulation(l Log, blockedStrings, allowedStrings []string) error {
	blocked, err := convertStrings(blockedStrings)
	if err != nil {
		return err
	}
	allowed, err := convertStrings(allowedStrings)
	if err != nil {
		return err
	}
	log := l.(*log)
	simulation(log, blocked, allowed)
	return nil
}

func (o *limiter) CompileBlockList(l Log, target, excessive int) ([]string, []string, error) {
	log := l.(*log)
	cidrsCount, ipsCount, unknown := o.getCidrCount(log)

	if ipsCount < excessive {
		return nil, unknown, ErrorNotExcessive{}
	}

	if excessive <= target {
		return nil, unknown, fmt.Errorf("(excessive = %d) <= (target = %d)", excessive, target)
	}

	blockedIPs := 0
	blockedIndex := 0
	blocked := make([]string, 0, 100)
	for ; blockedIndex < len(cidrsCount) && ipsCount-blockedIPs > target; blockedIndex++ {
		e := cidrsCount[blockedIndex]
		blockedIPs += e.count
		blocked = append(blocked, e.cidr.String())
	}

	var err error
	if ipsCount-blockedIPs > target {
		err = ErrorTargetNotReached{}
	}

	return blocked, unknown, err
}
