// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"testing"

	auth_model "code.gitea.io/gitea/models/auth"
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/gitrepo"
	"code.gitea.io/gitea/modules/graceful"
	repo_module "code.gitea.io/gitea/modules/repository"
	api "code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/tests"

	"github.com/stretchr/testify/require"
)

func TestRepoSSHSignedTags(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// Preparations
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	repo, _, f := CreateDeclarativeRepo(t, user, "", nil, nil, nil)
	defer f()

	// Set up an SSH key for the tagger
	tmpDir := t.TempDir()
	err := os.Chmod(tmpDir, 0o700)
	require.NoError(t, err)

	signingKey := fmt.Sprintf("%s/ssh_key", tmpDir)

	cmd := exec.Command("ssh-keygen", "-t", "ed25519", "-N", "", "-f", signingKey)
	err = cmd.Run()
	require.NoError(t, err)

	// Set up git config for the tagger
	_ = git.NewCommand(git.DefaultContext, "config", "user.name").AddDynamicArguments(user.Name).Run(&git.RunOpts{Dir: repo.RepoPath()})
	_ = git.NewCommand(git.DefaultContext, "config", "user.email").AddDynamicArguments(user.Email).Run(&git.RunOpts{Dir: repo.RepoPath()})
	_ = git.NewCommand(git.DefaultContext, "config", "gpg.format", "ssh").Run(&git.RunOpts{Dir: repo.RepoPath()})
	_ = git.NewCommand(git.DefaultContext, "config", "user.signingkey").AddDynamicArguments(signingKey).Run(&git.RunOpts{Dir: repo.RepoPath()})

	// Open the git repo
	gitRepo, _ := gitrepo.OpenRepository(git.DefaultContext, repo)
	defer gitRepo.Close()

	// Create a signed tag
	err = git.NewCommand(git.DefaultContext, "tag", "-s", "-m", "this is a signed tag", "ssh-signed-tag").Run(&git.RunOpts{Dir: repo.RepoPath()})
	require.NoError(t, err)

	// Sync the tag to the DB
	repo_module.SyncRepoTags(graceful.GetManager().ShutdownContext(), repo.ID)

	// Helper functions
	assertTagSignedStatus := func(t *testing.T, isSigned bool) {
		t.Helper()

		req := NewRequestf(t, "GET", "%s/releases/tag/ssh-signed-tag", repo.HTMLURL())
		resp := MakeRequest(t, req, http.StatusOK)
		doc := NewHTMLParser(t, resp.Body)

		doc.AssertElement(t, ".tag-signature-row .gitea-unlock", !isSigned)
		doc.AssertElement(t, ".tag-signature-row .gitea-lock", isSigned)
	}

	t.Run("unverified", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		assertTagSignedStatus(t, false)
	})

	t.Run("verified", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		// Upload the signing key
		keyData, err := os.ReadFile(fmt.Sprintf("%s.pub", signingKey))
		require.NoError(t, err)
		key := string(keyData)

		session := loginUser(t, user.Name)
		token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteUser)

		req := NewRequestWithJSON(t, "POST", "/api/v1/user/keys", &api.CreateKeyOption{
			Key:   key,
			Title: "test key",
		}).AddTokenAuth(token)
		resp := MakeRequest(t, req, http.StatusCreated)

		var pubkey *api.PublicKey
		DecodeJSON(t, resp, &pubkey)

		// Mark the key as verified
		db.GetEngine(db.DefaultContext).Exec("UPDATE `public_key` SET verified = true WHERE id = ?", pubkey.ID)

		// Check the tag page
		assertTagSignedStatus(t, true)
	})
}
