//go:build go1.24

// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package log

import "unsafe"

//go:linkname runtime_getProfLabel runtime/pprof.runtime_getProfLabel
func runtime_getProfLabel() unsafe.Pointer //nolint

// Struct definitions copied from: https://github.com/golang/go/blob/ca63101df47a4467bc80faa654fc19d68e583952/src/runtime/pprof/label.go
type label struct {
	key   string
	value string
}

type LabelSet struct {
	list []label
}

type labelMap struct {
	LabelSet
}

func getGoroutineLabels() map[string]string {
	l := (*labelMap)(runtime_getProfLabel())
	if l == nil {
		return nil
	}

	m := make(map[string]string, len(l.list))
	for _, label := range l.list {
		m[label.key] = label.value
	}
	return m
}
