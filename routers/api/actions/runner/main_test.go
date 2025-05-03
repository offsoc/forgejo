// Copyright 2024 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package runner

import (
	"testing"

	"forgejo.org/models/unittest"

	_ "forgejo.org/models/forgefed"
)

func TestMain(m *testing.M) {
	unittest.MainTest(m)
}
