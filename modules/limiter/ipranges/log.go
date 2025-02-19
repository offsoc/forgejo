// Copyright 2025 The Forgejo Authors.
// SPDX-License-Identifier: GPL-3.0-or-later

package ipranges

import (
	"cmp"
	"net/netip"
	"slices"
	"sync"
	"time"
)

type logEntry struct {
	ip      netip.Addr
	created time.Time
	blocked bool
}

type sortedBy string

const (
	sortedByTime = "time" // chronological
	sortedByIP   = "ip"   // increasing order
)

type log struct {
	mutex   RWLocker
	begin   time.Time
	entries *container[logEntry]
	sorted  sortedBy
}

func (o *log) init(capacity int) {
	o.entries = newContainer[logEntry](capacity)
	o.reset()
	o.mutex = &sync.RWMutex{}
	o.sorted = sortedByTime
}

func (o *log) reset() {
	o.entries.reset()
	o.begin = time.Now()
}

func (o *log) capacity() int {
	return o.entries.capacity()
}

func (o *log) add(a netip.Addr, blocked bool) {
	o.mutex.Lock()
	o.entries.push(logEntry{
		ip:      a,
		created: time.Now(),
		blocked: blocked,
	})
	o.mutex.Unlock()
}

const (
	copyReset   = true
	copyNoReset = false
)

func (o *log) copy(reset bool) *log {
	o.mutex.RLock()
	entries := o.entries.copy()
	if reset {
		o.entries.reset()
	}
	o.mutex.RUnlock()
	c := &log{
		entries: entries,
		begin:   o.begin,
	}
	return c
}

func (o *log) all() []logEntry {
	return o.entries.all()
}

func (o *log) sortByTime() {
	if o.sorted == sortedByTime {
		return
	}
	slices.SortFunc(o.all(), func(a, b logEntry) int {
		return cmp.Compare(b.created.Unix(), a.created.Unix())
	})

	o.sorted = sortedByTime
}

func (o *log) sortByIP() {
	if o.sorted == sortedByIP {
		return
	}
	slices.SortFunc(o.all(), func(a, b logEntry) int {
		if a.ip == b.ip {
			return 0
		}
		if a.ip.Less(b.ip) {
			return -1
		}
		return 1
	})

	o.sorted = sortedByIP
}

func uniqueIPs(entries []logEntry) []netip.Addr {
	unique := entriesByUniqueIP(entries)
	ips := make([]netip.Addr, 0, len(unique))
	for i := 1; i < len(unique); i++ {
		ips = append(ips, unique[i].ip)
	}
	return ips
}

func entriesByUniqueIP(entries []logEntry) []logEntry {
	unique := make([]logEntry, 0, 1000)
	for i := 1; i < len(entries); i++ {
		if entries[i].ip == entries[i-1].ip {
			continue
		}
		unique = append(unique, entries[i])
	}
	return unique
}
