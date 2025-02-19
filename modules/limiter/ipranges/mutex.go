// Copyright 2025 The Forgejo Authors.
// SPDX-License-Identifier: GPL-3.0-or-later

package ipranges

import (
	"sync"
)

type RWLocker interface {
	sync.Locker
	RLock()
	RUnlock()
}
