// Copyright 2024 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package generate

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeJwtSecret(t *testing.T) {
	_, err := DecodeJwtSecret("abcd")
	require.ErrorContains(t, err, "invalid base64 decoded length")
	_, err = DecodeJwtSecret(strings.Repeat("a", 64))
	require.ErrorContains(t, err, "invalid base64 decoded length")

	str32 := strings.Repeat("x", 32)
	encoded32 := base64.RawURLEncoding.EncodeToString([]byte(str32))
	decoded32, err := DecodeJwtSecret(encoded32)
	require.NoError(t, err)
	assert.Equal(t, str32, string(decoded32))
}

func TestNewJwtSecret(t *testing.T) {
	secret, encoded := NewJwtSecret()
	assert.Len(t, secret, 32)
	decoded, err := DecodeJwtSecret(encoded)
	require.NoError(t, err)
	assert.Equal(t, secret, decoded)
}
