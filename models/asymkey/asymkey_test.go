// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later
package asymkey

import (
	"testing"

	"forgejo.org/models/unittest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserHasAsymKey(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	t.Run("No key", func(t *testing.T) {
		ok, err := HasAsymKeyByUID(t.Context(), 1)
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("SSH key", func(t *testing.T) {
		ok, err := HasAsymKeyByUID(t.Context(), 2)
		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("GPG key", func(t *testing.T) {
		ok, err := HasAsymKeyByUID(t.Context(), 36)
		require.NoError(t, err)
		assert.True(t, ok)
	})
}
