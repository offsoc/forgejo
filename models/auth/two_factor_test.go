// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later
package auth

import (
	"testing"

	"forgejo.org/models/unittest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHasTwoFactorByUID(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	t.Run("No twofactor", func(t *testing.T) {
		ok, err := HasTwoFactorByUID(t.Context(), 2)
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("WebAuthn credential", func(t *testing.T) {
		ok, err := HasTwoFactorByUID(t.Context(), 32)
		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("TOTP", func(t *testing.T) {
		ok, err := HasTwoFactorByUID(t.Context(), 24)
		require.NoError(t, err)
		assert.True(t, ok)
	})
}
