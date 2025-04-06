// Copyright 2018 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"net/http"
	"testing"

	"forgejo.org/models/unittest"
	"forgejo.org/models/webhook"
	"forgejo.org/services/contexttest"

	"github.com/stretchr/testify/assert"
)

func TestTestHook(t *testing.T) {
	unittest.PrepareTestEnv(t)

	ctx, _ := contexttest.MockAPIContext(t, "user2/repo1/wiki/_pages")
	ctx.SetParams(":id", "1")
	contexttest.LoadUser(t, ctx, 2)
	contexttest.LoadRepo(t, ctx, 1)
	contexttest.LoadGitRepo(t, ctx)
	defer ctx.Repo.GitRepo.Close()
	contexttest.LoadRepoCommit(t, ctx)
	TestHook(ctx)
	assert.Equal(t, http.StatusNoContent, ctx.Resp.Status())

	unittest.AssertExistsAndLoadBean(t, &webhook.HookTask{
		HookID: 1,
	}, unittest.Cond("is_delivered=?", false))
}
