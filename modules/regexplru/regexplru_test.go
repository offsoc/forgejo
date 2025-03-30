// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package regexplru

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegexpLru(t *testing.T) {
	r, err := GetCompiled("a")
	require.NoError(t, err)
	assert.True(t, r.MatchString("a"))

	r, err = GetCompiled("a")
	require.NoError(t, err)
	assert.True(t, r.MatchString("a"))

	assert.Equal(t, 1, lruCache.Len())

	_, err = GetCompiled("(")
	require.Error(t, err)
	assert.Equal(t, 2, lruCache.Len())
}
