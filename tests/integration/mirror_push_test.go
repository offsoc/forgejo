// Copyright 2021 The Gitea Authors. All rights reserved.
// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	asymkey_model "forgejo.org/models/asymkey"
	"forgejo.org/models/db"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unit"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/git"
	"forgejo.org/modules/gitrepo"
	"forgejo.org/modules/optional"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/test"
	gitea_context "forgejo.org/services/context"
	doctor "forgejo.org/services/doctor"
	"forgejo.org/services/migrations"
	mirror_service "forgejo.org/services/mirror"
	repo_service "forgejo.org/services/repository"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMirrorPush(t *testing.T) {
	onGiteaRun(t, testMirrorPush)
}

func testMirrorPush(t *testing.T, u *url.URL) {
	defer tests.PrepareTestEnv(t)()
	defer test.MockVariableValue(&setting.Migrations.AllowLocalNetworks, true)()

	require.NoError(t, migrations.Init())

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	srcRepo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})

	mirrorRepo, err := repo_service.CreateRepositoryDirectly(db.DefaultContext, user, user, repo_service.CreateRepoOptions{
		Name: "test-push-mirror",
	})
	require.NoError(t, err)

	ctx := NewAPITestContext(t, user.LowerName, srcRepo.Name)

	doCreatePushMirror(ctx, fmt.Sprintf("%s%s/%s", u.String(), url.PathEscape(ctx.Username), url.PathEscape(mirrorRepo.Name)), user.LowerName, userPassword)(t)
	doCreatePushMirror(ctx, fmt.Sprintf("%s%s/%s", u.String(), url.PathEscape(ctx.Username), url.PathEscape("does-not-matter")), user.LowerName, userPassword)(t)

	mirrors, _, err := repo_model.GetPushMirrorsByRepoID(db.DefaultContext, srcRepo.ID, db.ListOptions{})
	require.NoError(t, err)
	assert.Len(t, mirrors, 2)

	ok := mirror_service.SyncPushMirror(t.Context(), mirrors[0].ID)
	assert.True(t, ok)

	srcGitRepo, err := gitrepo.OpenRepository(git.DefaultContext, srcRepo)
	require.NoError(t, err)
	defer srcGitRepo.Close()

	srcCommit, err := srcGitRepo.GetBranchCommit("master")
	require.NoError(t, err)

	mirrorGitRepo, err := gitrepo.OpenRepository(git.DefaultContext, mirrorRepo)
	require.NoError(t, err)
	defer mirrorGitRepo.Close()

	mirrorCommit, err := mirrorGitRepo.GetBranchCommit("master")
	require.NoError(t, err)

	assert.Equal(t, srcCommit.ID, mirrorCommit.ID)

	// Test that we can "repair" push mirrors where the remote doesn't exist in git's state.
	// To do that, we artificially remove the remote...
	cmd := git.NewCommand(db.DefaultContext, "remote", "rm").AddDynamicArguments(mirrors[0].RemoteName)
	_, _, err = cmd.RunStdString(&git.RunOpts{Dir: srcRepo.RepoPath()})
	require.NoError(t, err)

	// ...then ensure that trying to get its remote address fails
	_, err = repo_model.GetPushMirrorRemoteAddress(srcRepo.OwnerName, srcRepo.Name, mirrors[0].RemoteName)
	require.Error(t, err)

	// ...and that we can fix it.
	err = doctor.FixPushMirrorsWithoutGitRemote(db.DefaultContext, nil, true)
	require.NoError(t, err)

	// ...and after fixing, we only have one remote
	mirrors, _, err = repo_model.GetPushMirrorsByRepoID(db.DefaultContext, srcRepo.ID, db.ListOptions{})
	require.NoError(t, err)
	assert.Len(t, mirrors, 1)

	// ...one we can get the address of, and it's not the one we removed
	remoteAddress, err := repo_model.GetPushMirrorRemoteAddress(srcRepo.OwnerName, srcRepo.Name, mirrors[0].RemoteName)
	require.NoError(t, err)
	assert.Contains(t, remoteAddress, "does-not-matter")

	// Cleanup
	doRemovePushMirror(ctx, fmt.Sprintf("%s%s/%s", u.String(), url.PathEscape(ctx.Username), url.PathEscape(mirrorRepo.Name)), user.LowerName, userPassword, int(mirrors[0].ID))(t)
	mirrors, _, err = repo_model.GetPushMirrorsByRepoID(db.DefaultContext, srcRepo.ID, db.ListOptions{})
	require.NoError(t, err)
	assert.Empty(t, mirrors)
}

func doCreatePushMirror(ctx APITestContext, address, username, password string) func(t *testing.T) {
	return func(t *testing.T) {
		csrf := GetCSRF(t, ctx.Session, fmt.Sprintf("/%s/%s/settings", url.PathEscape(ctx.Username), url.PathEscape(ctx.Reponame)))

		req := NewRequestWithValues(t, "POST", fmt.Sprintf("/%s/%s/settings", url.PathEscape(ctx.Username), url.PathEscape(ctx.Reponame)), map[string]string{
			"_csrf":                csrf,
			"action":               "push-mirror-add",
			"push_mirror_address":  address,
			"push_mirror_username": username,
			"push_mirror_password": password,
			"push_mirror_interval": "0",
		})
		ctx.Session.MakeRequest(t, req, http.StatusSeeOther)

		flashCookie := ctx.Session.GetCookie(gitea_context.CookieNameFlash)
		assert.NotNil(t, flashCookie)
		assert.Contains(t, flashCookie.Value, "success")
	}
}

func doRemovePushMirror(ctx APITestContext, address, username, password string, pushMirrorID int) func(t *testing.T) {
	return func(t *testing.T) {
		csrf := GetCSRF(t, ctx.Session, fmt.Sprintf("/%s/%s/settings", url.PathEscape(ctx.Username), url.PathEscape(ctx.Reponame)))

		req := NewRequestWithValues(t, "POST", fmt.Sprintf("/%s/%s/settings", url.PathEscape(ctx.Username), url.PathEscape(ctx.Reponame)), map[string]string{
			"_csrf":                csrf,
			"action":               "push-mirror-remove",
			"push_mirror_id":       strconv.Itoa(pushMirrorID),
			"push_mirror_address":  address,
			"push_mirror_username": username,
			"push_mirror_password": password,
			"push_mirror_interval": "0",
		})
		ctx.Session.MakeRequest(t, req, http.StatusSeeOther)

		flashCookie := ctx.Session.GetCookie(gitea_context.CookieNameFlash)
		assert.NotNil(t, flashCookie)
		assert.Contains(t, flashCookie.Value, "success")
	}
}

func TestSSHPushMirror(t *testing.T) {
	_, err := exec.LookPath("ssh")
	if err != nil {
		t.Skip("SSH executable not present")
	}

	onGiteaRun(t, func(t *testing.T, _ *url.URL) {
		defer test.MockVariableValue(&setting.Migrations.AllowLocalNetworks, true)()
		defer test.MockVariableValue(&setting.Mirror.Enabled, true)()
		defer test.MockVariableValue(&setting.SSH.RootPath, t.TempDir())()
		require.NoError(t, migrations.Init())

		user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
		srcRepo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 2})
		assert.False(t, srcRepo.HasWiki())
		sess := loginUser(t, user.Name)
		pushToRepo, _, f := tests.CreateDeclarativeRepoWithOptions(t, user, tests.DeclarativeRepoOptions{
			Name:         optional.Some("push-mirror-test"),
			AutoInit:     optional.Some(false),
			EnabledUnits: optional.Some([]unit.Type{unit.TypeCode}),
		})
		defer f()

		sshURL := fmt.Sprintf("ssh://%s@%s/%s.git", setting.SSH.User, net.JoinHostPort(setting.SSH.ListenHost, strconv.Itoa(setting.SSH.ListenPort)), pushToRepo.FullName())
		t.Run("Mutual exclusive", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequestWithValues(t, "POST", fmt.Sprintf("/%s/settings", srcRepo.FullName()), map[string]string{
				"_csrf":                GetCSRF(t, sess, fmt.Sprintf("/%s/settings", srcRepo.FullName())),
				"action":               "push-mirror-add",
				"push_mirror_address":  sshURL,
				"push_mirror_username": "username",
				"push_mirror_password": "password",
				"push_mirror_use_ssh":  "true",
				"push_mirror_interval": "0",
			})
			resp := sess.MakeRequest(t, req, http.StatusOK)
			htmlDoc := NewHTMLParser(t, resp.Body)

			errMsg := htmlDoc.Find(".ui.negative.message").Text()
			assert.Contains(t, errMsg, "Cannot use public key and password based authentication in combination.")
		})

		inputSelector := `input[id="push_mirror_use_ssh"]`

		t.Run("SSH not available", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			defer test.MockVariableValue(&git.HasSSHExecutable, false)()

			req := NewRequestWithValues(t, "POST", fmt.Sprintf("/%s/settings", srcRepo.FullName()), map[string]string{
				"_csrf":                GetCSRF(t, sess, fmt.Sprintf("/%s/settings", srcRepo.FullName())),
				"action":               "push-mirror-add",
				"push_mirror_address":  sshURL,
				"push_mirror_use_ssh":  "true",
				"push_mirror_interval": "0",
			})
			resp := sess.MakeRequest(t, req, http.StatusOK)
			htmlDoc := NewHTMLParser(t, resp.Body)

			errMsg := htmlDoc.Find(".ui.negative.message").Text()
			assert.Contains(t, errMsg, "SSH authentication isn't available.")

			htmlDoc.AssertElement(t, inputSelector, false)
		})

		t.Run("SSH available", func(t *testing.T) {
			req := NewRequest(t, "GET", fmt.Sprintf("/%s/settings", srcRepo.FullName()))
			resp := sess.MakeRequest(t, req, http.StatusOK)

			htmlDoc := NewHTMLParser(t, resp.Body)
			htmlDoc.AssertElement(t, inputSelector, true)
		})

		t.Run("Normal", func(t *testing.T) {
			var pushMirror *repo_model.PushMirror
			t.Run("Adding", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				req := NewRequestWithValues(t, "POST", fmt.Sprintf("/%s/settings", srcRepo.FullName()), map[string]string{
					"_csrf":                GetCSRF(t, sess, fmt.Sprintf("/%s/settings", srcRepo.FullName())),
					"action":               "push-mirror-add",
					"push_mirror_address":  sshURL,
					"push_mirror_use_ssh":  "true",
					"push_mirror_interval": "0",
				})
				sess.MakeRequest(t, req, http.StatusSeeOther)

				flashCookie := sess.GetCookie(gitea_context.CookieNameFlash)
				assert.NotNil(t, flashCookie)
				assert.Contains(t, flashCookie.Value, "success")

				pushMirror = unittest.AssertExistsAndLoadBean(t, &repo_model.PushMirror{RepoID: srcRepo.ID})
				assert.NotEmpty(t, pushMirror.PrivateKey)
				assert.NotEmpty(t, pushMirror.PublicKey)
			})

			publickey := ""
			t.Run("Publickey", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				req := NewRequest(t, "GET", fmt.Sprintf("/%s/settings", srcRepo.FullName()))
				resp := sess.MakeRequest(t, req, http.StatusOK)
				htmlDoc := NewHTMLParser(t, resp.Body)

				publickey = htmlDoc.Find(".ui.table td a[data-clipboard-text]").AttrOr("data-clipboard-text", "")
				assert.Equal(t, publickey, pushMirror.GetPublicKey())
			})

			t.Run("Add deploy key", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				req := NewRequestWithValues(t, "POST", fmt.Sprintf("/%s/settings/keys", pushToRepo.FullName()), map[string]string{
					"_csrf":       GetCSRF(t, sess, fmt.Sprintf("/%s/settings/keys", pushToRepo.FullName())),
					"title":       "push mirror key",
					"content":     publickey,
					"is_writable": "true",
				})
				sess.MakeRequest(t, req, http.StatusSeeOther)

				unittest.AssertExistsAndLoadBean(t, &asymkey_model.DeployKey{Name: "push mirror key", RepoID: pushToRepo.ID})
			})

			t.Run("Synchronize", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				req := NewRequestWithValues(t, "POST", fmt.Sprintf("/%s/settings", srcRepo.FullName()), map[string]string{
					"_csrf":          GetCSRF(t, sess, fmt.Sprintf("/%s/settings", srcRepo.FullName())),
					"action":         "push-mirror-sync",
					"push_mirror_id": strconv.FormatInt(pushMirror.ID, 10),
				})
				sess.MakeRequest(t, req, http.StatusSeeOther)
			})

			t.Run("Check mirrored content", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()
				shortSHA := "1032bbf17f"

				req := NewRequest(t, "GET", fmt.Sprintf("/%s", srcRepo.FullName()))
				resp := sess.MakeRequest(t, req, http.StatusOK)
				htmlDoc := NewHTMLParser(t, resp.Body)

				assert.Contains(t, htmlDoc.Find(".shortsha").Text(), shortSHA)

				assert.Eventually(t, func() bool {
					req = NewRequest(t, "GET", fmt.Sprintf("/%s", pushToRepo.FullName()))
					resp = sess.MakeRequest(t, req, NoExpectedStatus)
					htmlDoc = NewHTMLParser(t, resp.Body)

					return resp.Code == http.StatusOK && htmlDoc.Find(".shortsha").Text() == shortSHA
				}, time.Second*30, time.Second)
			})

			t.Run("Check known host keys", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				knownHosts, err := os.ReadFile(filepath.Join(setting.SSH.RootPath, "known_hosts"))
				require.NoError(t, err)

				publicKey, err := os.ReadFile(setting.SSH.ServerHostKeys[0] + ".pub")
				require.NoError(t, err)

				assert.Contains(t, string(knownHosts), string(publicKey))
			})
		})
	})
}

func TestPushMirrorSettings(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		defer test.MockVariableValue(&setting.Migrations.AllowLocalNetworks, true)()
		defer test.MockVariableValue(&setting.Mirror.Enabled, true)()
		require.NoError(t, migrations.Init())

		user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
		srcRepo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 2})
		srcRepo2 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 3})
		assert.False(t, srcRepo.HasWiki())
		sess := loginUser(t, user.Name)
		pushToRepo, _, f := tests.CreateDeclarativeRepoWithOptions(t, user, tests.DeclarativeRepoOptions{
			Name:         optional.Some("push-mirror-test"),
			AutoInit:     optional.Some(false),
			EnabledUnits: optional.Some([]unit.Type{unit.TypeCode}),
		})
		defer f()

		t.Run("Adding", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequestWithValues(t, "POST", fmt.Sprintf("/%s/settings", srcRepo2.FullName()), map[string]string{
				"_csrf":                GetCSRF(t, sess, fmt.Sprintf("/%s/settings", srcRepo2.FullName())),
				"action":               "push-mirror-add",
				"push_mirror_address":  u.String() + pushToRepo.FullName(),
				"push_mirror_interval": "0",
			})
			sess.MakeRequest(t, req, http.StatusSeeOther)

			req = NewRequestWithValues(t, "POST", fmt.Sprintf("/%s/settings", srcRepo.FullName()), map[string]string{
				"_csrf":                GetCSRF(t, sess, fmt.Sprintf("/%s/settings", srcRepo.FullName())),
				"action":               "push-mirror-add",
				"push_mirror_address":  u.String() + pushToRepo.FullName(),
				"push_mirror_interval": "0",
			})
			sess.MakeRequest(t, req, http.StatusSeeOther)

			flashCookie := sess.GetCookie(gitea_context.CookieNameFlash)
			assert.NotNil(t, flashCookie)
			assert.Contains(t, flashCookie.Value, "success")
		})

		mirrors, _, err := repo_model.GetPushMirrorsByRepoID(db.DefaultContext, srcRepo.ID, db.ListOptions{})
		require.NoError(t, err)
		assert.Len(t, mirrors, 1)
		mirrorID := mirrors[0].ID

		mirrors, _, err = repo_model.GetPushMirrorsByRepoID(db.DefaultContext, srcRepo2.ID, db.ListOptions{})
		require.NoError(t, err)
		assert.Len(t, mirrors, 1)

		t.Run("Interval", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			unittest.AssertExistsAndLoadBean(t, &repo_model.PushMirror{ID: mirrorID - 1})

			req := NewRequestWithValues(t, "POST", fmt.Sprintf("/%s/settings", srcRepo.FullName()), map[string]string{
				"_csrf":                GetCSRF(t, sess, fmt.Sprintf("/%s/settings", srcRepo.FullName())),
				"action":               "push-mirror-update",
				"push_mirror_id":       strconv.FormatInt(mirrorID-1, 10),
				"push_mirror_interval": "10m0s",
			})
			sess.MakeRequest(t, req, http.StatusNotFound)

			req = NewRequestWithValues(t, "POST", fmt.Sprintf("/%s/settings", srcRepo.FullName()), map[string]string{
				"_csrf":                GetCSRF(t, sess, fmt.Sprintf("/%s/settings", srcRepo.FullName())),
				"action":               "push-mirror-update",
				"push_mirror_id":       strconv.FormatInt(mirrorID, 10),
				"push_mirror_interval": "10m0s",
			})
			sess.MakeRequest(t, req, http.StatusSeeOther)

			flashCookie := sess.GetCookie(gitea_context.CookieNameFlash)
			assert.NotNil(t, flashCookie)
			assert.Contains(t, flashCookie.Value, "success")
		})
	})
}
