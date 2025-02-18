// Copyright 2025 The Forgejo Authors.
// SPDX-License-Identifier: GPL-3.0-or-later

package limiter_test

import (
	"context"
	"testing"

	"code.gitea.io/gitea/models/limiter"
	"code.gitea.io/gitea/models/unittest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIPRangesSettings(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	ctx := context.Background()

	i, err := limiter.NewIPRanges(ctx)
	require.NoError(t, err)

	assert.NoError(t, i.SetEnabled(ctx, true))
	assert.True(t, i.GetEnabled(ctx))
	i.SetEnabled(ctx, false)
	assert.False(t, i.GetEnabled(ctx))

	expectedIPCount := 123
	require.NoError(t, i.SetExpectedIPCount(ctx, expectedIPCount))
	assert.Equal(t, expectedIPCount, i.GetExpectedIPCount(ctx))

	excessiveIPCount := 574
	require.NoError(t, i.SetExcessiveIPCount(ctx, excessiveIPCount))
	assert.Equal(t, excessiveIPCount, i.GetExcessiveIPCount(ctx))

	blockTop := 43
	require.NoError(t, i.SetBlockTop(ctx, blockTop))
	assert.Equal(t, blockTop, i.GetBlockTop(ctx))

	periodicity := "@daily"
	require.NoError(t, i.SetPeriodicity(ctx, periodicity))
	assert.Equal(t, periodicity, i.GetPeriodicity(ctx))
}

func TestIPRangeList(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	ctx := context.Background()

	i, err := limiter.NewIPRanges(ctx)
	require.NoError(t, err)

	ipranges := []string{"1.2.3.0/24", "6.7.0.0/16"}

	for n := 0; n < 2; n++ {
		// do it twice to verify a new list overrides the previous one
		require.NoError(t, i.SetBlocked(ctx, ipranges))
	}
	actual, err := i.GetBlocked(ctx)
	require.NoError(t, err)
	assert.EqualValues(t, ipranges, actual)

	for n := 0; n < 2; n++ {
		// do it twice to verify a new list overrides the previous one
		require.NoError(t, i.SetAllowed(ctx, ipranges))
	}
	actual, err = i.GetAllowed(ctx)
	require.NoError(t, err)
	assert.EqualValues(t, ipranges, actual)
}
