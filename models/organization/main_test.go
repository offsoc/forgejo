// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package organization_test

import (
	"testing"

	"forgejo.org/models/unittest"

	_ "forgejo.org/models"
	_ "forgejo.org/models/actions"
	_ "forgejo.org/models/activities"
	_ "forgejo.org/models/forgefed"
	_ "forgejo.org/models/organization"
	_ "forgejo.org/models/repo"
	_ "forgejo.org/models/user"
)

func TestMain(m *testing.M) {
	unittest.MainTest(m)
}
