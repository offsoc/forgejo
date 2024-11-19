// Copyright 2023 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package updatechecker

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDNSUpdate(t *testing.T) {
	version, err := getVersionDNS("release.forgejo.org")
	require.NoError(t, err)
	assert.NotEmpty(t, version)
}
