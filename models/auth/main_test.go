// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package auth_test

import (
	"testing"

	"forgejo.org/models/unittest"

	_ "forgejo.org/models"
	_ "forgejo.org/models/actions"
	_ "forgejo.org/models/activities"
	_ "forgejo.org/models/auth"
	_ "forgejo.org/models/forgefed"
	_ "forgejo.org/models/perm/access"
)

func TestMain(m *testing.M) {
	unittest.MainTest(m)
}
