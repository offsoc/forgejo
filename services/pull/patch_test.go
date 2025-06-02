// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package pull

import (
	"fmt"
	"testing"

	issues_model "forgejo.org/models/issues"
	"forgejo.org/models/unittest"
	"forgejo.org/modules/setting"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadHeadRevision(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	t.Run("AGit", func(t *testing.T) {
		t.Run("New", func(t *testing.T) {
			ctx := &testPatchContext{}
			require.NoError(t, ctx.LoadHeadRevision(t.Context(), &issues_model.PullRequest{Flow: issues_model.PullRequestFlowAGit, HeadCommitID: "Commit!"}))

			assert.Empty(t, ctx.env)
			assert.Equal(t, "Commit!", ctx.headRev)
			assert.True(t, ctx.headIsCommitID)
		})
		t.Run("Existing", func(t *testing.T) {
			ctx := &testPatchContext{}
			require.NoError(t, ctx.LoadHeadRevision(t.Context(), &issues_model.PullRequest{Flow: issues_model.PullRequestFlowAGit, Index: 371}))

			assert.Empty(t, ctx.env)
			assert.Equal(t, "refs/pull/371/head", ctx.headRev)
			assert.False(t, ctx.headIsCommitID)
		})
	})

	t.Run("Same repository", func(t *testing.T) {
		ctx := &testPatchContext{}
		require.NoError(t, ctx.LoadHeadRevision(t.Context(), unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 1})))

		assert.Empty(t, ctx.env)
		assert.Equal(t, "refs/heads/branch1", ctx.headRev)
		assert.False(t, ctx.headIsCommitID)
	})

	t.Run("Across repository", func(t *testing.T) {
		pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 3})
		require.NoError(t, pr.LoadHeadRepo(t.Context()))

		ctx := &testPatchContext{}
		require.NoError(t, ctx.LoadHeadRevision(t.Context(), pr))

		if assert.NotEmpty(t, ctx.env) {
			assert.Equal(t, fmt.Sprintf("GIT_ALTERNATE_OBJECT_DIRECTORIES=%s/user13/repo11.git/objects", setting.RepoRootPath), ctx.env[len(ctx.env)-1])
		}
		assert.Equal(t, "0abcb056019adb8336cf9db3ad9d9cf80cd4b141", ctx.headRev)
		assert.True(t, ctx.headIsCommitID)
	})
}
