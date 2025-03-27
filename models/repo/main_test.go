// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo_test

import (
	"testing"

	"forgejo.org/models/unittest"

	_ "forgejo.org/models" // register table model
	_ "forgejo.org/models/actions"
	_ "forgejo.org/models/activities"
	_ "forgejo.org/models/forgefed"
	_ "forgejo.org/models/perm/access" // register table model
	_ "forgejo.org/models/repo"        // register table model
	_ "forgejo.org/models/user"        // register table model
)

func TestMain(m *testing.M) {
	unittest.MainTest(m)
}
