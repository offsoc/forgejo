//go:build pam

// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package pam

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPamAuth(t *testing.T) {
	result, err := Auth("gitea", "user1", "false-pwd")
	require.Error(t, err)
	assert.EqualError(t, err, "Authentication failure")
	assert.Len(t, result, 0)
}
