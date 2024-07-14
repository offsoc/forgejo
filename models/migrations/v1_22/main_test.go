// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package v1_22 //nolint

import (
	"testing"

	migration_tests "code.gitea.io/gitea/models/migrations/test"
)

func TestMain(m *testing.M) {
	migration_tests.MainTest(m)
}
