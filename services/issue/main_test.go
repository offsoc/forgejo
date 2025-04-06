// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issue

import (
	"testing"

	"forgejo.org/models/unittest"
	"forgejo.org/modules/setting"
	"forgejo.org/services/webhook"

	_ "forgejo.org/models/actions"
)

func TestMain(m *testing.M) {
	unittest.MainTest(m, &unittest.TestOptions{
		SetUp: func() error {
			setting.LoadQueueSettings()
			return webhook.Init()
		},
	})
}
