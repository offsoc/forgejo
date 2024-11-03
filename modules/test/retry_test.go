// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRetry(t *testing.T) {
	assert.True(t, Retry(func() bool { return true }, 1))
	i := 0
	fail3 := func() bool {
		i++
		return i >= 3
	}
	assert.True(t, Retry(fail3, 5))
	assert.Equal(t, 3, i)
	i = 0
	assert.False(t, Retry(fail3, 2))
}
