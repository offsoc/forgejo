// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/tests"

	"github.com/stretchr/testify/require"
)

func TestGistHttpPush(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		defer tests.PrepareTestEnv(t)()

		tempDir := t.TempDir()

		cmd := git.NewCommand(db.DefaultContext, "clone")
		cmd.AddDynamicArguments(fmt.Sprintf("%sgists/df852aec.git", u.String()))
		cmd.AddDynamicArguments(tempDir)

		require.NoError(t, cmd.Run(&git.RunOpts{}))

		require.NoError(t, os.WriteFile(filepath.Join(tempDir, "new.txt"), []byte("New text"), 0o644))

		require.NoError(t, git.AddChanges(tempDir, true))
		require.NoError(t, git.CommitChanges(tempDir, git.CommitChangesOptions{Message: "Test"}))

		require.NoError(t, git.NewCommand(db.DefaultContext, "push").Run(&git.RunOpts{}))
	})
}
