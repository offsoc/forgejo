// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package base

import (
	"testing"

	migrations_tests "code.gitea.io/gitea/models/migrations/test"
)

func TestMain(m *testing.M) {
	migrations_tests.MainTest(m)
}
