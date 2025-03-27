// Copyright 2024 The Forgejo Authors
// SPDX-License-Identifier: MIT

package actions

import (
	"testing"

	"forgejo.org/models/unittest"

	_ "forgejo.org/models/actions"
	_ "forgejo.org/models/activities"
	_ "forgejo.org/models/forgefed"
)

func TestMain(m *testing.M) {
	unittest.MainTest(m)
}
