// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package system_test

import (
	"testing"

	"forgejo.org/models/unittest"

	_ "forgejo.org/models" // register models
	_ "forgejo.org/models/actions"
	_ "forgejo.org/models/activities"
	_ "forgejo.org/models/forgefed"
	_ "forgejo.org/models/system" // register models of system
)

func TestMain(m *testing.M) {
	unittest.MainTest(m)
}
