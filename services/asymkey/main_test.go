// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package asymkey

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
