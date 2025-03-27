// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package convert

import (
	"testing"

	"forgejo.org/models/unittest"

	_ "forgejo.org/models/actions"
	_ "forgejo.org/models/forgefed"
)

func TestMain(m *testing.M) {
	unittest.MainTest(m)
}
