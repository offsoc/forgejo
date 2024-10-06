// Copyright 2023 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgejo_migrations //nolint:revive

import (
	"testing"

	migration_tests "code.gitea.io/gitea/models/migrations/test"
)

func TestMain(m *testing.M) {
	migration_tests.MainTest(m)
}
