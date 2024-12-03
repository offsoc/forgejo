// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package templates_test

import (
	"context"
	"testing"

	"code.gitea.io/gitea/models/unittest"
	"code.gitea.io/gitea/modules/markup"

	_ "code.gitea.io/gitea/models"
	_ "code.gitea.io/gitea/models/forgefed"
	_ "code.gitea.io/gitea/models/issues"
)

func TestMain(m *testing.M) {
	markup.Init(&markup.ProcessorHelper{
		IsUsernameMentionable: func(ctx context.Context, username string) bool {
			return username == "mention-user"
		},
	})
	unittest.MainTest(m)
}
