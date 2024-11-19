// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package git

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// This file contains utility functions that are used across multiple tests,
// but not in production code.

func skipIfSHA256NotSupported(t *testing.T) {
	if CheckGitVersionAtLeast("2.42") != nil {
		t.Skip("skipping because installed Git version doesn't support SHA256")
	}
}

func TestHashFilePathForWebUI(t *testing.T) {
	assert.Equal(t,
		"8843d7f92416211de9ebb963ff4ce28125932878",
		HashFilePathForWebUI("foobar"),
	)
}
