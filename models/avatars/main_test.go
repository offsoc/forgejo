// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package avatars_test

import (
	"testing"

	"forgejo.org/models/unittest"

	_ "forgejo.org/models"
	_ "forgejo.org/models/activities"
	_ "forgejo.org/models/perm/access"
)

func TestMain(m *testing.M) {
	unittest.MainTest(m)
}
