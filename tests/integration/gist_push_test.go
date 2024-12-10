// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	auth_model "code.gitea.io/gitea/models/auth"
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/modules/git"

	"github.com/stretchr/testify/require"
)

func TestGistsPushHttp(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		tempDir := t.TempDir()

		require.NoError(t, git.Clone(db.DefaultContext, fmt.Sprintf("%sgists/df852aec.git", u.String()), tempDir, git.CloneRepoOptions{}))

		require.NoError(t, os.WriteFile(filepath.Join(tempDir, "new.txt"), []byte("New text"), 0o644))

		require.NoError(t, git.AddChanges(tempDir, true))
		require.NoError(t, git.CommitChanges(tempDir, git.CommitChangesOptions{Message: "Test"}))

		// The push should fail without login
		require.Error(t, git.NewCommand(db.DefaultContext, "push").Run(&git.RunOpts{Dir: tempDir}))

		// the push should succeed with login
		cmd := git.NewCommand(db.DefaultContext, "push")
		cmd.AddDynamicArguments(fmt.Sprintf("%s://user2:password@%s/gists/df852aec.git", u.Scheme, u.Host))
		require.NoError(t, cmd.Run(&git.RunOpts{Dir: tempDir}))
	})
}

func TestGistsPushSSH(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		withKeyFile(t, "test-gist-key", func(keyFile string) {
			ctx := NewAPITestContext(t, "user2", "", auth_model.AccessTokenScopeWriteUser)

			t.Run("CreateUserKey", doAPICreateUserKey(ctx, "test-key-gist", keyFile))

			tempDir := t.TempDir()

			require.NoError(t, git.Clone(db.DefaultContext, createSSHUrl("gists/df852aec.git", u).String(), tempDir, git.CloneRepoOptions{}))

			require.NoError(t, os.WriteFile(filepath.Join(tempDir, "new.txt"), []byte("New text"), 0o644))

			require.NoError(t, git.AddChanges(tempDir, true))
			require.NoError(t, git.CommitChanges(tempDir, git.CommitChangesOptions{Message: "Test"}))

			require.NoError(t, git.NewCommand(db.DefaultContext, "push").Run(&git.RunOpts{Dir: tempDir}))
		})
	})
}
