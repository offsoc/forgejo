// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package access_test

import (
	"testing"

	"forgejo.org/models/unittest"

	_ "forgejo.org/models"
	_ "forgejo.org/models/actions"
	_ "forgejo.org/models/activities"
	_ "forgejo.org/models/forgefed"
	_ "forgejo.org/models/repo"
	_ "forgejo.org/models/user"
)

func TestMain(m *testing.M) {
	unittest.MainTest(m)
}
