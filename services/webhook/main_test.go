// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package webhook

import (
	"testing"

	"forgejo.org/models/unittest"
	"forgejo.org/modules/hostmatcher"
	"forgejo.org/modules/setting"

	_ "forgejo.org/models"
	_ "forgejo.org/models/actions"
	_ "forgejo.org/models/forgefed"
)

func TestMain(m *testing.M) {
	// for tests, allow only loopback IPs
	setting.Webhook.AllowedHostList = hostmatcher.MatchBuiltinLoopback
	unittest.MainTest(m, &unittest.TestOptions{
		SetUp: func() error {
			setting.LoadQueueSettings()
			return Init()
		},
	})
}
