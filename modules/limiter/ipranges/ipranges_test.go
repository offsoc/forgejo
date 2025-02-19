// Copyright 2025 The Forgejo Authors.
// SPDX-License-Identifier: GPL-3.0-or-later

package ipranges

import (
	"net/netip"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLimiterSetGet(t *testing.T) {
	blocked := []string{"1.2.3.0/24", "5.6.7.0/10"}
	allowed := []string{"1.2.3.0/24", "5.6.7.0/10"}

	l := New()
	assert.NoError(t, l.Init())
	assert.NoError(t, l.SetBlockList(blocked))
	assert.EqualValues(t, blocked, l.GetBlockList())
	require.NoError(t, l.SetAllowList(allowed))
	assert.EqualValues(t, allowed, l.GetAllowList())
	maxIPs := 200
	l.SetMaxIPs(maxIPs)
	assert.EqualValues(t, maxIPs, l.GetMaxIPs())
}

type testRWMutex struct {
	called     int
	wouldblock bool
	locked     bool
	rlocked    bool
}

func (o *testRWMutex) Lock() {
	o.called++
	if o.locked || o.rlocked {
		o.wouldblock = true
	}
	o.locked = true
}

func (o *testRWMutex) Unlock() {}

func (o *testRWMutex) RLock() {
	o.called++
	if o.locked {
		o.wouldblock = true
	}
	o.rlocked = true
}

func (o *testRWMutex) RUnlock() {}

func TestLimiterBlockedOrAllowedMutex(t *testing.T) {
	l := (&limiter{}).initLocks()

	for _, testCase := range []struct {
		name       string
		wouldblock bool
		lock       bool
		rlock      bool
		fun        func()
	}{
		{name: "RLock does not block GetBlockList", wouldblock: false, lock: false, rlock: true, fun: func() { l.GetBlockList() }},
		{name: "Lock blocks GetBlockList", wouldblock: true, lock: true, rlock: false, fun: func() { l.GetBlockList() }},
		{name: "RLock blocks SetBlockList", wouldblock: true, lock: false, rlock: true, fun: func() { l.SetBlockList([]string{}) }},
		{name: "Lock blocks SetBlockList", wouldblock: true, lock: true, rlock: false, fun: func() { l.SetBlockList([]string{}) }},

		{name: "RLock does not block GetAllowList", wouldblock: false, lock: false, rlock: true, fun: func() { l.GetAllowList() }},
		{name: "Lock blocks GetAllowList", wouldblock: true, lock: true, rlock: false, fun: func() { l.GetBlockList() }},
		{name: "RLock blocks SetAllowList", wouldblock: true, lock: false, rlock: true, fun: func() { l.SetAllowList([]string{}) }},
		{name: "Lock blocks SetAllowList", wouldblock: true, lock: true, rlock: false, fun: func() { l.SetAllowList([]string{}) }},

		{name: "RLock does not block allow", wouldblock: false, lock: false, rlock: true, fun: func() { l.allow(netip.Addr{}) }},
		{name: "Lock blocks allow", wouldblock: true, lock: true, rlock: false, fun: func() { l.allow(netip.Addr{}) }},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			m := &testRWMutex{}
			l.blockedOrAllowedMutex = m
			if testCase.lock {
				m.Lock()
			}
			if testCase.rlock {
				m.RLock()
			}
			testCase.fun()
			assert.Equal(t, 2, m.called)
			assert.Equal(t, testCase.wouldblock, m.wouldblock)
		})
	}
}

func TestLimiterIPSMutex(t *testing.T) {
	l := (&limiter{}).initLocks()
	require.NoError(t, l.Init())

	for _, testCase := range []struct {
		name       string
		wouldblock bool
		lock       bool
		rlock      bool
		fun        func()
	}{
		{name: "RLock does not block MostUsedCidrs", wouldblock: false, lock: false, rlock: true, fun: func() { l.MostUsedCidrs(0) }},
		{name: "Lock blocks MostUsedCidrs", wouldblock: true, lock: true, rlock: false, fun: func() { l.MostUsedCidrs(0) }},

		{name: "RLock blocks add", wouldblock: true, lock: false, rlock: true, fun: func() { l.add("1.1.1.1") }},
		{name: "Lock blocks add", wouldblock: true, lock: true, rlock: false, fun: func() { l.add("1.1.1.1") }},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			m := &testRWMutex{}
			l.ipsMutex = m
			if testCase.lock {
				m.Lock()
			}
			if testCase.rlock {
				m.RLock()
			}
			testCase.fun()
			assert.Equal(t, 2, m.called)
			assert.Equal(t, testCase.wouldblock, m.wouldblock)
		})
	}
}

func TestLimiterFind(t *testing.T) {
	l := []netip.Prefix{
		netip.MustParsePrefix("1.2.3.0/24"),
		netip.MustParsePrefix("1.4.3.0/24"),
		netip.MustParsePrefix("10.5.3.0/12"),
	}

	for _, testCase := range []struct {
		name string
		has  bool
		n    int
		ip   string
	}{
		{name: "before first range", has: false, ip: "1.1.0.0"},
		{name: "first of first range", has: true, n: 0, ip: "1.2.3.0"},
		{name: "in first range", has: true, n: 0, ip: "1.2.3.1"},
		{name: "last of first range", has: true, n: 0, ip: "1.2.3.255"},
		{name: "between first and second range", has: false, ip: "1.3.0.1"},
		{name: "in second range", has: true, n: 1, ip: "1.4.3.10"},
		{name: "after last range", has: false, ip: "200.4.1.10"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			a, err := netip.ParseAddr(testCase.ip)
			require.NoError(t, err)
			n, has := find(l, a)
			assert.Equal(t, testCase.has, has)
			if has {
				assert.Equal(t, testCase.n, n)
			}
		})
	}
}

func TestLimiterAdd(t *testing.T) {
	l := (&limiter{}).initLocks()
	maxiIPs := 3
	l.SetMaxIPs(maxiIPs)
	last := "1.1.1.4"
	for _, ip := range []string{
		"1.1.1.1",
		"1.1.1.2",
		"1.1.1.3",
		last,
	} {
		a, err := l.add(ip)
		require.NoError(t, err)
		assert.Equal(t, ip, a.String())
	}
	// start from the first when it overflows
	assert.Equal(t, last, l.ips[0].String())
}

func TestLimiterAddAndAllow(t *testing.T) {
	blocked0 := "1.2.0.0/16"
	blocked1 := "5.6.7.0/10"
	blocked := []string{
		blocked0,
		blocked1,
	}
	allowed := []string{
		"1.2.1.0/24",
		"7.8.9.0/16",
	}

	l := New()
	assert.NoError(t, l.Init())
	assert.NoError(t, l.SetBlockList(blocked))
	assert.NoError(t, l.SetAllowList(allowed))
	l.SetMaxIPs(200)

	for _, testCase := range []struct {
		name   string
		allow  bool
		ip     string
		reason string
	}{
		{name: "match and blocked", allow: false, ip: "1.2.0.1", reason: blocked0 + " 0"},
		{name: "match allowed has precedence over blocked", allow: true, ip: "1.2.1.1"},
		{name: "no match is allowed", allow: true, ip: "50.10.20.30"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			allow, reason, err := l.AddAndAllow(testCase.ip)
			require.NoError(t, err)
			assert.Equal(t, testCase.allow, allow)
			assert.Equal(t, testCase.reason, reason)
		})
	}
}

type testIPRanges struct {
	ipranges []netip.Prefix
}

func (o *testIPRanges) Load() error         { return nil }
func (o *testIPRanges) Get() []netip.Prefix { return o.ipranges }

func TestMostUsedCidrs(t *testing.T) {
	l := (&limiter{}).initLocks()
	first := "1.2.3.0/24"
	second := "1.4.3.0/24"
	third := "10.5.0.0/16"
	l.ipranges = &testIPRanges{
		ipranges: []netip.Prefix{
			netip.MustParsePrefix(first),
			netip.MustParsePrefix(second),
			netip.MustParsePrefix(third),
		},
	}

	for _, testCase := range []struct {
		name    string
		ips     []netip.Addr
		counts  []CidrCount
		unknown []string
	}{
		{
			name: "",
			ips: []netip.Addr{
				// first
				netip.MustParseAddr("1.2.3.1"),
				netip.MustParseAddr("1.2.3.1"),
				netip.MustParseAddr("1.2.3.1"),
				netip.MustParseAddr("1.2.3.2"),

				// unknown CIDR
				netip.MustParseAddr("1.3.0.2"),

				// second
				netip.MustParseAddr("1.4.3.2"),
				netip.MustParseAddr("1.4.3.2"),

				// third
				netip.MustParseAddr("10.5.0.2"),
				netip.MustParseAddr("10.5.0.3"),
				netip.MustParseAddr("10.5.0.4"),
				netip.MustParseAddr("10.5.0.4"),

				// unknown CIDR
				netip.MustParseAddr("20.10.0.5"),
			},
			counts: []CidrCount{
				{Cidr: third, Count: 3},
				{Cidr: first, Count: 2},
			},
			unknown: []string{
				"1.3.0.2",
				"20.10.0.5",
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			l.ips = testCase.ips
			counts, unknown := l.MostUsedCidrs(2)
			assert.Equal(t, testCase.counts, counts)
			assert.Equal(t, testCase.unknown, unknown)
		})
	}
}
