// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package test

import (
	"time"
)

func Retry(fun func() bool, tries int) bool {
	for i := 0; i < tries; i++ {
		if fun() {
			return true
		}
		<-time.After(1 * time.Second)
	}
	return false
}
