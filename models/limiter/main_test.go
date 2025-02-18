// Copyright 2025 The Forgejo Authors.
// SPDX-License-Identifier: GPL-3.0-or-later

package limiter

import (
	"testing"

	"code.gitea.io/gitea/models/unittest"
)

func TestMain(m *testing.M) {
	unittest.MainTest(m, &unittest.TestOptions{
		FixtureFiles: []string{
			"TestLimiter/limiter_ipranges_settings.yml",
			"TestLimiter/limiter_iprange_list.yml",
		},
	})
}
