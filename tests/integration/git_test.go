// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	auth_model "forgejo.org/models/auth"
	"forgejo.org/models/db"
	git_model "forgejo.org/models/git"
	issues_model "forgejo.org/models/issues"
	"forgejo.org/models/perm"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/git"
	"forgejo.org/modules/gitrepo"
	"forgejo.org/modules/lfs"
	"forgejo.org/modules/setting"
	api "forgejo.org/modules/structs"
	gitea_context "forgejo.org/services/context"
	files_service "forgejo.org/services/repository/files"
	"forgejo.org/tests"

	"github.com/kballard/go-shellquote"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	littleSize = 1024             // 1KiB
	bigSize    = 32 * 1024 * 1024 // 32MiB
)

func TestGit(t *testing.T) {
	onGiteaRun(t, testGit)
}

func testGit(t *testing.T, u *url.URL) {
	username := "user2"
	baseAPITestContext := NewAPITestContext(t, username, "repo1", auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)

	u.Path = baseAPITestContext.GitPath()

	forkedUserCtx := NewAPITestContext(t, "user4", "repo1", auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)

	t.Run("HTTP", func(t *testing.T) {
		ensureAnonymousClone(t, u)
		forEachObjectFormat(t, func(t *testing.T, objectFormat git.ObjectFormat) {
			defer tests.PrintCurrentTest(t)()
			httpContext := baseAPITestContext
			httpContext.Reponame = "repo-tmp-17-" + objectFormat.Name()
			forkedUserCtx.Reponame = httpContext.Reponame

			dstPath := t.TempDir()

			t.Run("CreateRepoInDifferentUser", doAPICreateRepository(forkedUserCtx, false, objectFormat))
			t.Run("AddUserAsCollaborator", doAPIAddCollaborator(forkedUserCtx, httpContext.Username, perm.AccessModeRead))

			t.Run("ForkFromDifferentUser", doAPIForkRepository(httpContext, forkedUserCtx.Username))

			u.Path = httpContext.GitPath()
			u.User = url.UserPassword(username, userPassword)

			t.Run("Clone", doGitClone(dstPath, u))

			dstPath2 := t.TempDir()

			t.Run("Partial Clone", doPartialGitClone(dstPath2, u))

			little, big := standardCommitAndPushTest(t, dstPath)
			littleLFS, bigLFS := lfsCommitAndPushTest(t, dstPath)
			rawTest(t, &httpContext, little, big, littleLFS, bigLFS)
			mediaTest(t, &httpContext, little, big, littleLFS, bigLFS)

			t.Run("CreateAgitFlowPull", doCreateAgitFlowPull(dstPath, &httpContext, "test/head"))
			t.Run("InternalReferences", doInternalReferences(&httpContext, dstPath))
			t.Run("BranchProtect", doBranchProtect(&httpContext, dstPath))
			t.Run("AutoMerge", doAutoPRMerge(&httpContext, dstPath))
			t.Run("CreatePRAndSetManuallyMerged", doCreatePRAndSetManuallyMerged(httpContext, httpContext, dstPath, "master", "test-manually-merge"))
			t.Run("MergeFork", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()
				t.Run("CreatePRAndMerge", doMergeFork(httpContext, forkedUserCtx, "master", httpContext.Username+":master"))
				rawTest(t, &forkedUserCtx, little, big, littleLFS, bigLFS)
				mediaTest(t, &forkedUserCtx, little, big, littleLFS, bigLFS)
			})

			t.Run("PushCreate", doPushCreate(httpContext, u, objectFormat))
		})
	})
	t.Run("SSH", func(t *testing.T) {
		forEachObjectFormat(t, func(t *testing.T, objectFormat git.ObjectFormat) {
			defer tests.PrintCurrentTest(t)()
			sshContext := baseAPITestContext
			sshContext.Reponame = "repo-tmp-18-" + objectFormat.Name()
			keyname := "my-testing-key"
			forkedUserCtx.Reponame = sshContext.Reponame
			t.Run("CreateRepoInDifferentUser", doAPICreateRepository(forkedUserCtx, false, objectFormat))
			t.Run("AddUserAsCollaborator", doAPIAddCollaborator(forkedUserCtx, sshContext.Username, perm.AccessModeRead))
			t.Run("ForkFromDifferentUser", doAPIForkRepository(sshContext, forkedUserCtx.Username))

			// Setup key the user ssh key
			withKeyFile(t, keyname, func(keyFile string) {
				var publicKeyID int64
				t.Run("CreateUserKey", doAPICreateUserKey(sshContext, "test-key-"+objectFormat.Name(), keyFile, func(t *testing.T, pk api.PublicKey) {
					publicKeyID = pk.ID
				}))

				// Setup remote link
				// TODO: get url from api
				sshURL := createSSHUrl(sshContext.GitPath(), u)

				// Setup clone folder
				dstPath := t.TempDir()

				t.Run("Clone", doGitClone(dstPath, sshURL))

				little, big := standardCommitAndPushTest(t, dstPath)
				littleLFS, bigLFS := lfsCommitAndPushTest(t, dstPath)
				rawTest(t, &sshContext, little, big, littleLFS, bigLFS)
				mediaTest(t, &sshContext, little, big, littleLFS, bigLFS)

				t.Run("CreateAgitFlowPull", doCreateAgitFlowPull(dstPath, &sshContext, "test/head2"))
				t.Run("InternalReferences", doInternalReferences(&sshContext, dstPath))
				t.Run("BranchProtect", doBranchProtect(&sshContext, dstPath))
				t.Run("MergeFork", func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()
					t.Run("CreatePRAndMerge", doMergeFork(sshContext, forkedUserCtx, "master", sshContext.Username+":master"))
					rawTest(t, &forkedUserCtx, little, big, littleLFS, bigLFS)
					mediaTest(t, &forkedUserCtx, little, big, littleLFS, bigLFS)
				})

				t.Run("PushCreate", doPushCreate(sshContext, sshURL, objectFormat))
				t.Run("LFS no access", doLFSNoAccess(sshContext, publicKeyID, objectFormat))
			})
		})
	})
}

func ensureAnonymousClone(t *testing.T, u *url.URL) {
	dstLocalPath := t.TempDir()
	t.Run("CloneAnonymous", doGitClone(dstLocalPath, u))
}

func standardCommitAndPushTest(t *testing.T, dstPath string) (little, big string) {
	t.Run("Standard", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		little, big = commitAndPushTest(t, dstPath, "data-file-")
	})
	return little, big
}

func lfsCommitAndPushTest(t *testing.T, dstPath string) (littleLFS, bigLFS string) {
	t.Run("LFS", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		prefix := "lfs-data-file-"
		err := git.NewCommand(git.DefaultContext, "lfs").AddArguments("install").Run(&git.RunOpts{Dir: dstPath})
		require.NoError(t, err)
		_, _, err = git.NewCommand(git.DefaultContext, "lfs").AddArguments("track").AddDynamicArguments(prefix + "*").RunStdString(&git.RunOpts{Dir: dstPath})
		require.NoError(t, err)
		err = git.AddChanges(dstPath, false, ".gitattributes")
		require.NoError(t, err)

		err = git.CommitChangesWithArgs(dstPath, git.AllowLFSFiltersArgs(), git.CommitChangesOptions{
			Committer: &git.Signature{
				Email: "user2@example.com",
				Name:  "User Two",
				When:  time.Now(),
			},
			Author: &git.Signature{
				Email: "user2@example.com",
				Name:  "User Two",
				When:  time.Now(),
			},
			Message: fmt.Sprintf("Testing commit @ %v", time.Now()),
		})
		require.NoError(t, err)

		littleLFS, bigLFS = commitAndPushTest(t, dstPath, prefix)

		t.Run("Locks", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			lockTest(t, dstPath)
		})
	})
	return littleLFS, bigLFS
}

func commitAndPushTest(t *testing.T, dstPath, prefix string) (little, big string) {
	t.Run("PushCommit", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		t.Run("Little", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			little = doCommitAndPush(t, littleSize, dstPath, prefix)
		})
		t.Run("Big", func(t *testing.T) {
			if testing.Short() {
				t.Skip("Skipping test in short mode.")
				return
			}
			defer tests.PrintCurrentTest(t)()
			big = doCommitAndPush(t, bigSize, dstPath, prefix)
		})
	})
	return little, big
}

func rawTest(t *testing.T, ctx *APITestContext, little, big, littleLFS, bigLFS string) {
	t.Run("Raw", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		username := ctx.Username
		reponame := ctx.Reponame

		session := loginUser(t, username)

		// Request raw paths
		req := NewRequest(t, "GET", path.Join("/", username, reponame, "/raw/branch/master/", little))
		resp := session.MakeRequestNilResponseRecorder(t, req, http.StatusOK)
		assert.Equal(t, littleSize, resp.Length)

		if setting.LFS.StartServer {
			req = NewRequest(t, "GET", path.Join("/", username, reponame, "/raw/branch/master/", littleLFS))
			resp := session.MakeRequest(t, req, http.StatusOK)
			assert.NotEqual(t, littleSize, resp.Body.Len())
			assert.LessOrEqual(t, resp.Body.Len(), 1024)
			if resp.Body.Len() != littleSize && resp.Body.Len() <= 1024 {
				assert.Contains(t, resp.Body.String(), lfs.MetaFileIdentifier)
			}
		}

		if !testing.Short() {
			req = NewRequest(t, "GET", path.Join("/", username, reponame, "/raw/branch/master/", big))
			resp := session.MakeRequestNilResponseRecorder(t, req, http.StatusOK)
			assert.Equal(t, bigSize, resp.Length)

			if setting.LFS.StartServer {
				req = NewRequest(t, "GET", path.Join("/", username, reponame, "/raw/branch/master/", bigLFS))
				resp := session.MakeRequest(t, req, http.StatusOK)
				assert.NotEqual(t, bigSize, resp.Body.Len())
				if resp.Body.Len() != bigSize && resp.Body.Len() <= 1024 {
					assert.Contains(t, resp.Body.String(), lfs.MetaFileIdentifier)
				}
			}
		}
	})
}

func mediaTest(t *testing.T, ctx *APITestContext, little, big, littleLFS, bigLFS string) {
	t.Run("Media", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		username := ctx.Username
		reponame := ctx.Reponame

		session := loginUser(t, username)

		// Request media paths
		req := NewRequest(t, "GET", path.Join("/", username, reponame, "/media/branch/master/", little))
		resp := session.MakeRequestNilResponseRecorder(t, req, http.StatusOK)
		assert.Equal(t, littleSize, resp.Length)

		req = NewRequest(t, "GET", path.Join("/", username, reponame, "/media/branch/master/", littleLFS))
		resp = session.MakeRequestNilResponseRecorder(t, req, http.StatusOK)
		assert.Equal(t, littleSize, resp.Length)

		if !testing.Short() {
			req = NewRequest(t, "GET", path.Join("/", username, reponame, "/media/branch/master/", big))
			resp = session.MakeRequestNilResponseRecorder(t, req, http.StatusOK)
			assert.Equal(t, bigSize, resp.Length)

			if setting.LFS.StartServer {
				req = NewRequest(t, "GET", path.Join("/", username, reponame, "/media/branch/master/", bigLFS))
				resp = session.MakeRequestNilResponseRecorder(t, req, http.StatusOK)
				assert.Equal(t, bigSize, resp.Length)
			}
		}
	})
}

func lockTest(t *testing.T, repoPath string) {
	lockFileTest(t, "README.md", repoPath)
}

func lockFileTest(t *testing.T, filename, repoPath string) {
	_, _, err := git.NewCommand(git.DefaultContext, "lfs").AddArguments("locks").RunStdString(&git.RunOpts{Dir: repoPath})
	require.NoError(t, err)
	_, _, err = git.NewCommand(git.DefaultContext, "lfs").AddArguments("lock").AddDynamicArguments(filename).RunStdString(&git.RunOpts{Dir: repoPath})
	require.NoError(t, err)
	_, _, err = git.NewCommand(git.DefaultContext, "lfs").AddArguments("locks").RunStdString(&git.RunOpts{Dir: repoPath})
	require.NoError(t, err)
	_, _, err = git.NewCommand(git.DefaultContext, "lfs").AddArguments("unlock").AddDynamicArguments(filename).RunStdString(&git.RunOpts{Dir: repoPath})
	require.NoError(t, err)
}

func doCommitAndPush(t *testing.T, size int64, repoPath, prefix string) string {
	name := generateCommitWithNewData(t, size, repoPath, "user2@example.com", "User Two", prefix)
	_, _, err := git.NewCommand(git.DefaultContext, "push", "origin", "master").RunStdString(&git.RunOpts{Dir: repoPath}) // Push
	require.NoError(t, err)
	return name
}

func generateCommitWithNewData(t *testing.T, size int64, repoPath, email, fullName, prefix string) string {
	t.Helper()
	tmpFile, err := os.CreateTemp(repoPath, prefix)
	require.NoError(t, err)
	defer tmpFile.Close()
	_, err = io.CopyN(tmpFile, rand.Reader, size)
	require.NoError(t, err)

	// Commit
	// Now here we should explicitly allow lfs filters to run
	globalArgs := git.AllowLFSFiltersArgs()
	require.NoError(t, git.AddChangesWithArgs(repoPath, globalArgs, false, filepath.Base(tmpFile.Name())))
	require.NoError(t, git.CommitChangesWithArgs(repoPath, globalArgs, git.CommitChangesOptions{
		Committer: &git.Signature{
			Email: email,
			Name:  fullName,
			When:  time.Now(),
		},
		Author: &git.Signature{
			Email: email,
			Name:  fullName,
			When:  time.Now(),
		},
		Message: fmt.Sprintf("Testing commit @ %v", time.Now()),
	}))
	return filepath.Base(tmpFile.Name())
}

func doBranchProtect(baseCtx *APITestContext, dstPath string) func(t *testing.T) {
	return func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		t.Run("CreateBranchProtected", doGitCreateBranch(dstPath, "protected"))
		t.Run("PushProtectedBranch", doGitPushTestRepository(dstPath, "origin", "protected"))

		ctx := NewAPITestContext(t, baseCtx.Username, baseCtx.Reponame, auth_model.AccessTokenScopeWriteRepository)

		t.Run("PushToNewProtectedBranch", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			t.Run("CreateBranchProtected", doGitCreateBranch(dstPath, "before-create-1"))
			t.Run("ProtectProtectedBranch", doProtectBranch(ctx, "before-create-1", parameterProtectBranch{
				"enable_push":     "all",
				"apply_to_admins": "on",
			}))
			t.Run("PushProtectedBranch", doGitPushTestRepository(dstPath, "origin", "before-create-1"))

			t.Run("GenerateCommit", func(t *testing.T) {
				generateCommitWithNewData(t, littleSize, dstPath, "user2@example.com", "User Two", "protected-file-data-")
			})

			t.Run("ProtectProtectedBranch", doProtectBranch(ctx, "before-create-2", parameterProtectBranch{
				"enable_push":             "all",
				"protected_file_patterns": "protected-file-data-*",
				"apply_to_admins":         "on",
			}))

			doGitPushTestRepositoryFail(dstPath, "origin", "HEAD:before-create-2")(t)
		})

		t.Run("FailToPushToProtectedBranch", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			t.Run("ProtectProtectedBranch", doProtectBranch(ctx, "protected"))
			t.Run("Create modified-protected-branch", doGitCheckoutBranch(dstPath, "-b", "modified-protected-branch", "protected"))
			t.Run("GenerateCommit", func(t *testing.T) {
				generateCommitWithNewData(t, littleSize, dstPath, "user2@example.com", "User Two", "branch-data-file-")
			})

			doGitPushTestRepositoryFail(dstPath, "origin", "modified-protected-branch:protected")(t)
		})

		t.Run("PushToUnprotectedBranch", doGitPushTestRepository(dstPath, "origin", "modified-protected-branch:unprotected"))

		t.Run("FailToPushProtectedFilesToProtectedBranch", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			t.Run("Create modified-protected-file-protected-branch", doGitCheckoutBranch(dstPath, "-b", "modified-protected-file-protected-branch", "protected"))
			t.Run("GenerateCommit", func(t *testing.T) {
				generateCommitWithNewData(t, littleSize, dstPath, "user2@example.com", "User Two", "protected-file-")
			})

			t.Run("ProtectedFilePathsApplyToAdmins", doProtectBranch(ctx, "protected"))
			doGitPushTestRepositoryFail(dstPath, "origin", "modified-protected-file-protected-branch:protected")(t)

			doGitCheckoutBranch(dstPath, "protected")(t)
			doGitPull(dstPath, "origin", "protected")(t)
		})

		t.Run("PushUnprotectedFilesToProtectedBranch", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			t.Run("Create modified-unprotected-file-protected-branch", doGitCheckoutBranch(dstPath, "-b", "modified-unprotected-file-protected-branch", "protected"))
			t.Run("UnprotectedFilePaths", doProtectBranch(ctx, "protected", parameterProtectBranch{
				"unprotected_file_patterns": "unprotected-file-*",
			}))
			t.Run("GenerateCommit", func(t *testing.T) {
				generateCommitWithNewData(t, littleSize, dstPath, "user2@example.com", "User Two", "unprotected-file-")
			})
			doGitPushTestRepository(dstPath, "origin", "modified-unprotected-file-protected-branch:protected")(t)
			doGitCheckoutBranch(dstPath, "protected")(t)
			doGitPull(dstPath, "origin", "protected")(t)
		})

		user, err := user_model.GetUserByName(db.DefaultContext, baseCtx.Username)
		require.NoError(t, err)
		t.Run("WhitelistUsers", doProtectBranch(ctx, "protected", parameterProtectBranch{
			"enable_push":      "whitelist",
			"enable_whitelist": "on",
			"whitelist_users":  strconv.FormatInt(user.ID, 10),
		}))

		t.Run("WhitelistedUserFailToForcePushToProtectedBranch", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			t.Run("Create toforce", doGitCheckoutBranch(dstPath, "-b", "toforce", "master"))
			t.Run("GenerateCommit", func(t *testing.T) {
				generateCommitWithNewData(t, littleSize, dstPath, "user2@example.com", "User Two", "branch-data-file-")
			})
			doGitPushTestRepositoryFail(dstPath, "-f", "origin", "toforce:protected")(t)
		})

		t.Run("WhitelistedUserPushToProtectedBranch", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			t.Run("Create topush", doGitCheckoutBranch(dstPath, "-b", "topush", "protected"))
			t.Run("GenerateCommit", func(t *testing.T) {
				generateCommitWithNewData(t, littleSize, dstPath, "user2@example.com", "User Two", "branch-data-file-")
			})
			doGitPushTestRepository(dstPath, "origin", "topush:protected")(t)
		})
	}
}

type parameterProtectBranch map[string]string

func doProtectBranch(ctx APITestContext, branch string, addParameter ...parameterProtectBranch) func(t *testing.T) {
	// We are going to just use the owner to set the protection.
	return func(t *testing.T) {
		repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{Name: ctx.Reponame, OwnerName: ctx.Username})
		rule := &git_model.ProtectedBranch{RuleName: branch, RepoID: repo.ID}
		unittest.LoadBeanIfExists(rule)

		csrf := GetCSRF(t, ctx.Session, fmt.Sprintf("/%s/%s/settings/branches", url.PathEscape(ctx.Username), url.PathEscape(ctx.Reponame)))

		parameter := parameterProtectBranch{
			"_csrf":     csrf,
			"rule_id":   strconv.FormatInt(rule.ID, 10),
			"rule_name": branch,
		}
		if len(addParameter) > 0 {
			for k, v := range addParameter[0] {
				parameter[k] = v
			}
		}

		// Change branch to protected
		req := NewRequestWithValues(t, "POST", fmt.Sprintf("/%s/%s/settings/branches/edit", url.PathEscape(ctx.Username), url.PathEscape(ctx.Reponame)), parameter)
		ctx.Session.MakeRequest(t, req, http.StatusSeeOther)
		// Check if master branch has been locked successfully
		flashCookie := ctx.Session.GetCookie(gitea_context.CookieNameFlash)
		assert.NotNil(t, flashCookie)
		assert.Equal(t, "success%3DBranch%2Bprotection%2Bfor%2Brule%2B%2522"+url.QueryEscape(branch)+"%2522%2Bhas%2Bbeen%2Bupdated.", flashCookie.Value)
	}
}

func doMergeFork(ctx, baseCtx APITestContext, baseBranch, headBranch string) func(t *testing.T) {
	return func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		var pr api.PullRequest
		var err error

		// Create a test pull request
		t.Run("CreatePullRequest", func(t *testing.T) {
			pr, err = doAPICreatePullRequest(ctx, baseCtx.Username, baseCtx.Reponame, baseBranch, headBranch)(t)
			require.NoError(t, err)
		})

		// Ensure the PR page works.
		// For the base repository owner, the PR is not editable (maintainer edits are not enabled):
		t.Run("EnsureCanSeePull", doEnsureCanSeePull(baseCtx, pr, false))
		// For the head repository owner, the PR is editable:
		headSession := loginUser(t, "user2")
		headToken := getTokenForLoggedInUser(t, headSession, auth_model.AccessTokenScopeReadRepository, auth_model.AccessTokenScopeReadUser)
		headCtx := APITestContext{
			Session:  headSession,
			Token:    headToken,
			Username: baseCtx.Username,
			Reponame: baseCtx.Reponame,
		}
		t.Run("EnsureCanSeePull", doEnsureCanSeePull(headCtx, pr, true))

		// Confirm that there is no AGit Label
		// TODO: Refactor and move this check to a function
		t.Run("AGitLabelIsMissing", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			session := loginUser(t, ctx.Username)

			req := NewRequest(t, "GET", fmt.Sprintf("/%s/%s/pulls/%d", baseCtx.Username, baseCtx.Reponame, pr.Index))
			resp := session.MakeRequest(t, req, http.StatusOK)
			htmlDoc := NewHTMLParser(t, resp.Body)
			htmlDoc.AssertElement(t, "#agit-label", false)
		})

		// Then get the diff string
		var diffHash string
		var diffLength int
		t.Run("GetDiff", func(t *testing.T) {
			req := NewRequest(t, "GET", fmt.Sprintf("/%s/%s/pulls/%d.diff", url.PathEscape(baseCtx.Username), url.PathEscape(baseCtx.Reponame), pr.Index))
			resp := ctx.Session.MakeRequestNilResponseHashSumRecorder(t, req, http.StatusOK)
			diffHash = string(resp.Hash.Sum(nil))
			diffLength = resp.Length
		})

		// Now: Merge the PR & make sure that doesn't break the PR page or change its diff
		t.Run("MergePR", doAPIMergePullRequest(baseCtx, baseCtx.Username, baseCtx.Reponame, pr.Index))
		// for both users the PR is still visible but not editable anymore after it was merged
		t.Run("EnsureCanSeePull", doEnsureCanSeePull(baseCtx, pr, false))
		t.Run("EnsureCanSeePull", doEnsureCanSeePull(headCtx, pr, false))
		t.Run("CheckPR", func(t *testing.T) {
			oldMergeBase := pr.MergeBase
			pr2, err := doAPIGetPullRequest(baseCtx, baseCtx.Username, baseCtx.Reponame, pr.Index)(t)
			require.NoError(t, err)
			assert.Equal(t, oldMergeBase, pr2.MergeBase)
		})
		t.Run("EnsurDiffNoChange", doEnsureDiffNoChange(baseCtx, pr, diffHash, diffLength))

		// Then: Delete the head branch & make sure that doesn't break the PR page or change its diff
		t.Run("DeleteHeadBranch", doBranchDelete(baseCtx, baseCtx.Username, baseCtx.Reponame, headBranch))
		t.Run("EnsureCanSeePull", doEnsureCanSeePull(baseCtx, pr, false))
		t.Run("EnsureDiffNoChange", doEnsureDiffNoChange(baseCtx, pr, diffHash, diffLength))

		// Delete the head repository & make sure that doesn't break the PR page or change its diff
		t.Run("DeleteHeadRepository", doAPIDeleteRepository(ctx))
		t.Run("EnsureCanSeePull", doEnsureCanSeePull(baseCtx, pr, false))
		t.Run("EnsureDiffNoChange", doEnsureDiffNoChange(baseCtx, pr, diffHash, diffLength))
	}
}

func doCreatePRAndSetManuallyMerged(ctx, baseCtx APITestContext, dstPath, baseBranch, headBranch string) func(t *testing.T) {
	return func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		var (
			pr           api.PullRequest
			err          error
			lastCommitID string
		)

		trueBool := true
		falseBool := false

		t.Run("AllowSetManuallyMergedAndSwitchOffAutodetectManualMerge", doAPIEditRepository(baseCtx, &api.EditRepoOption{
			HasPullRequests:       &trueBool,
			AllowManualMerge:      &trueBool,
			AutodetectManualMerge: &falseBool,
		}))

		t.Run("CreateHeadBranch", doGitCreateBranch(dstPath, headBranch))
		t.Run("PushToHeadBranch", doGitPushTestRepository(dstPath, "origin", headBranch))
		t.Run("CreateEmptyPullRequest", func(t *testing.T) {
			pr, err = doAPICreatePullRequest(ctx, baseCtx.Username, baseCtx.Reponame, baseBranch, headBranch)(t)
			require.NoError(t, err)
		})
		lastCommitID = pr.Base.Sha
		t.Run("ManuallyMergePR", doAPIManuallyMergePullRequest(ctx, baseCtx.Username, baseCtx.Reponame, lastCommitID, pr.Index))
	}
}

func doEnsureCanSeePull(ctx APITestContext, pr api.PullRequest, editable bool) func(t *testing.T) {
	return func(t *testing.T) {
		req := NewRequest(t, "GET", fmt.Sprintf("/%s/%s/pulls/%d", url.PathEscape(ctx.Username), url.PathEscape(ctx.Reponame), pr.Index))
		ctx.Session.MakeRequest(t, req, http.StatusOK)
		req = NewRequest(t, "GET", fmt.Sprintf("/%s/%s/pulls/%d/files", url.PathEscape(ctx.Username), url.PathEscape(ctx.Reponame), pr.Index))
		resp := ctx.Session.MakeRequest(t, req, http.StatusOK)
		doc := NewHTMLParser(t, resp.Body)
		editButtonCount := doc.doc.Find("div.diff-file-header-actions a[href*='/_edit/']").Length()
		if editable {
			assert.Positive(t, editButtonCount, "Expected to find a button to edit a file in the PR diff view but there were none")
		} else {
			assert.Equal(t, 0, editButtonCount, "Expected not to find any buttons to edit files in PR diff view but there were some")
		}
		req = NewRequest(t, "GET", fmt.Sprintf("/%s/%s/pulls/%d/commits", url.PathEscape(ctx.Username), url.PathEscape(ctx.Reponame), pr.Index))
		ctx.Session.MakeRequest(t, req, http.StatusOK)
	}
}

func doEnsureDiffNoChange(ctx APITestContext, pr api.PullRequest, diffHash string, diffLength int) func(t *testing.T) {
	return func(t *testing.T) {
		req := NewRequest(t, "GET", fmt.Sprintf("/%s/%s/pulls/%d.diff", url.PathEscape(ctx.Username), url.PathEscape(ctx.Reponame), pr.Index))
		resp := ctx.Session.MakeRequestNilResponseHashSumRecorder(t, req, http.StatusOK)
		actual := string(resp.Hash.Sum(nil))
		actualLength := resp.Length

		equal := diffHash == actual
		assert.True(t, equal, "Unexpected change in the diff string: expected hash: %s size: %d but was actually: %s size: %d", hex.EncodeToString([]byte(diffHash)), diffLength, hex.EncodeToString([]byte(actual)), actualLength)
	}
}

func doPushCreate(ctx APITestContext, u *url.URL, objectFormat git.ObjectFormat) func(t *testing.T) {
	return func(t *testing.T) {
		if objectFormat == git.Sha256ObjectFormat {
			t.Skipf("push-create not supported for %s, see https://codeberg.org/forgejo/forgejo/issues/3783", objectFormat)
		}
		defer tests.PrintCurrentTest(t)()

		// create a context for a currently non-existent repository
		ctx.Reponame = fmt.Sprintf("repo-tmp-push-create-%s", u.Scheme)
		u.Path = ctx.GitPath()

		// Create a temporary directory
		tmpDir := t.TempDir()

		// Now create local repository to push as our test and set its origin
		t.Run("InitTestRepository", doGitInitTestRepository(tmpDir, objectFormat))
		t.Run("AddRemote", doGitAddRemote(tmpDir, "origin", u))

		// Disable "Push To Create" and attempt to push
		setting.Repository.EnablePushCreateUser = false
		t.Run("FailToPushAndCreateTestRepository", doGitPushTestRepositoryFail(tmpDir, "origin", "master"))

		// Enable "Push To Create"
		setting.Repository.EnablePushCreateUser = true

		// Assert that cloning from a non-existent repository does not create it and that it definitely wasn't create above
		t.Run("FailToCloneFromNonExistentRepository", doGitCloneFail(u))

		// Then "Push To Create"x
		t.Run("SuccessfullyPushAndCreateTestRepository", doGitPushTestRepository(tmpDir, "origin", "master"))

		// Finally, fetch repo from database and ensure the correct repository has been created
		repo, err := repo_model.GetRepositoryByOwnerAndName(db.DefaultContext, ctx.Username, ctx.Reponame)
		require.NoError(t, err)
		assert.False(t, repo.IsEmpty)
		assert.True(t, repo.IsPrivate)

		// Now add a remote that is invalid to "Push To Create"
		invalidCtx := ctx
		invalidCtx.Reponame = fmt.Sprintf("invalid/repo-tmp-push-create-%s", u.Scheme)
		u.Path = invalidCtx.GitPath()
		t.Run("AddInvalidRemote", doGitAddRemote(tmpDir, "invalid", u))

		// Fail to "Push To Create" the invalid
		t.Run("FailToPushAndCreateInvalidTestRepository", doGitPushTestRepositoryFail(tmpDir, "invalid", "master"))
	}
}

func doBranchDelete(ctx APITestContext, owner, repo, branch string) func(*testing.T) {
	return func(t *testing.T) {
		csrf := GetCSRF(t, ctx.Session, fmt.Sprintf("/%s/%s/branches", url.PathEscape(owner), url.PathEscape(repo)))

		req := NewRequestWithValues(t, "POST", fmt.Sprintf("/%s/%s/branches/delete?name=%s", url.PathEscape(owner), url.PathEscape(repo), url.QueryEscape(branch)), map[string]string{
			"_csrf": csrf,
		})
		ctx.Session.MakeRequest(t, req, http.StatusOK)
	}
}

func doAutoPRMerge(baseCtx *APITestContext, dstPath string) func(t *testing.T) {
	return func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		ctx := NewAPITestContext(t, baseCtx.Username, baseCtx.Reponame, auth_model.AccessTokenScopeWriteRepository)

		t.Run("CheckoutProtected", doGitCheckoutBranch(dstPath, "protected"))
		t.Run("PullProtected", doGitPull(dstPath, "origin", "protected"))
		t.Run("GenerateCommit", func(t *testing.T) {
			generateCommitWithNewData(t, littleSize, dstPath, "user2@example.com", "User Two", "branch-data-file-")
		})
		t.Run("PushToUnprotectedBranch", doGitPushTestRepository(dstPath, "origin", "protected:unprotected3"))
		var pr api.PullRequest
		var err error
		t.Run("CreatePullRequest", func(t *testing.T) {
			pr, err = doAPICreatePullRequest(ctx, baseCtx.Username, baseCtx.Reponame, "protected", "unprotected3")(t)
			require.NoError(t, err)
		})

		// Request repository commits page
		req := NewRequest(t, "GET", fmt.Sprintf("/%s/%s/pulls/%d/commits", baseCtx.Username, baseCtx.Reponame, pr.Index))
		resp := ctx.Session.MakeRequest(t, req, http.StatusOK)
		doc := NewHTMLParser(t, resp.Body)

		// Get first commit URL
		commitURL, exists := doc.doc.Find("#commits-table tbody tr td.sha a").Last().Attr("href")
		assert.True(t, exists)
		assert.NotEmpty(t, commitURL)

		commitID := path.Base(commitURL)

		addCommitStatus := func(status api.CommitStatusState) func(*testing.T) {
			return doAPICreateCommitStatus(ctx, commitID, api.CreateStatusOption{
				State:       status,
				TargetURL:   "http://test.ci/",
				Description: "",
				Context:     "testci",
			})
		}

		// Call API to add Pending status for commit
		t.Run("CreateStatus", addCommitStatus(api.CommitStatusPending))

		// Cancel not existing auto merge
		ctx.ExpectedCode = http.StatusNotFound
		t.Run("CancelAutoMergePR", doAPICancelAutoMergePullRequest(ctx, baseCtx.Username, baseCtx.Reponame, pr.Index))

		// Add auto merge request
		ctx.ExpectedCode = http.StatusCreated
		t.Run("AutoMergePR", doAPIAutoMergePullRequest(ctx, baseCtx.Username, baseCtx.Reponame, pr.Index))

		// Can not create schedule twice
		ctx.ExpectedCode = http.StatusConflict
		t.Run("AutoMergePRTwice", doAPIAutoMergePullRequest(ctx, baseCtx.Username, baseCtx.Reponame, pr.Index))

		// Cancel auto merge request
		ctx.ExpectedCode = http.StatusNoContent
		t.Run("CancelAutoMergePR", doAPICancelAutoMergePullRequest(ctx, baseCtx.Username, baseCtx.Reponame, pr.Index))

		// Add auto merge request
		ctx.ExpectedCode = http.StatusCreated
		t.Run("AutoMergePR", doAPIAutoMergePullRequest(ctx, baseCtx.Username, baseCtx.Reponame, pr.Index))

		// Check pr status
		ctx.ExpectedCode = 0
		pr, err = doAPIGetPullRequest(ctx, baseCtx.Username, baseCtx.Reponame, pr.Index)(t)
		require.NoError(t, err)
		assert.False(t, pr.HasMerged)

		// Call API to add Failure status for commit
		t.Run("CreateStatus", addCommitStatus(api.CommitStatusFailure))

		// Check pr status
		pr, err = doAPIGetPullRequest(ctx, baseCtx.Username, baseCtx.Reponame, pr.Index)(t)
		require.NoError(t, err)
		assert.False(t, pr.HasMerged)

		// Call API to add Success status for commit
		t.Run("CreateStatus", addCommitStatus(api.CommitStatusSuccess))

		// test pr status
		pr, err = doAPIGetPullRequest(ctx, baseCtx.Username, baseCtx.Reponame, pr.Index)(t)
		require.NoError(t, err)
		assert.True(t, pr.HasMerged)
	}
}

func doInternalReferences(ctx *APITestContext, dstPath string) func(t *testing.T) {
	return func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{OwnerName: ctx.Username, Name: ctx.Reponame})
		pr1 := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{HeadRepoID: repo.ID})

		_, stdErr, gitErr := git.NewCommand(git.DefaultContext, "push", "origin").AddDynamicArguments(fmt.Sprintf(":refs/pull/%d/head", pr1.Index)).RunStdString(&git.RunOpts{Dir: dstPath})
		require.Error(t, gitErr)
		assert.Contains(t, stdErr, fmt.Sprintf("remote: Forgejo: The deletion of refs/pull/%d/head is skipped as it's an internal reference.", pr1.Index))
		assert.Contains(t, stdErr, fmt.Sprintf("[remote rejected] refs/pull/%d/head (hook declined)", pr1.Index))

		_, stdErr, gitErr = git.NewCommand(git.DefaultContext, "push", "origin", "--force").AddDynamicArguments(fmt.Sprintf("HEAD~1:refs/pull/%d/head", pr1.Index)).RunStdString(&git.RunOpts{Dir: dstPath})
		require.Error(t, gitErr)
		assert.Contains(t, stdErr, fmt.Sprintf("remote: Forgejo: The modification of refs/pull/%d/head is skipped as it's an internal reference.", pr1.Index))
		assert.Contains(t, stdErr, fmt.Sprintf("[remote rejected] HEAD~1 -> refs/pull/%d/head (hook declined)", pr1.Index))
	}
}

func doCreateAgitFlowPull(dstPath string, ctx *APITestContext, headBranch string) func(t *testing.T) {
	return func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		// skip this test if git version is low
		if git.CheckGitVersionAtLeast("2.29") != nil {
			return
		}

		gitRepo, err := git.OpenRepository(git.DefaultContext, dstPath)
		require.NoError(t, err)

		defer gitRepo.Close()

		var (
			pr1, pr2 *issues_model.PullRequest
			commit   string
		)
		repo, err := repo_model.GetRepositoryByOwnerAndName(db.DefaultContext, ctx.Username, ctx.Reponame)
		require.NoError(t, err)

		pullNum := unittest.GetCount(t, &issues_model.PullRequest{})

		t.Run("CreateHeadBranch", doGitCreateBranch(dstPath, headBranch))

		t.Run("AddCommit", func(t *testing.T) {
			err := os.WriteFile(path.Join(dstPath, "test_file"), []byte("## test content"), 0o666)
			require.NoError(t, err)

			err = git.AddChanges(dstPath, true)
			require.NoError(t, err)

			err = git.CommitChanges(dstPath, git.CommitChangesOptions{
				Committer: &git.Signature{
					Email: "user2@example.com",
					Name:  "user2",
					When:  time.Now(),
				},
				Author: &git.Signature{
					Email: "user2@example.com",
					Name:  "user2",
					When:  time.Now(),
				},
				Message: "Testing commit 1",
			})
			require.NoError(t, err)
			commit, err = gitRepo.GetRefCommitID("HEAD")
			require.NoError(t, err)
		})

		t.Run("Push", func(t *testing.T) {
			err := git.NewCommand(git.DefaultContext, "push", "origin", "HEAD:refs/for/master", "-o").AddDynamicArguments("topic=" + headBranch).Run(&git.RunOpts{Dir: dstPath})
			require.NoError(t, err)

			unittest.AssertCount(t, &issues_model.PullRequest{}, pullNum+1)
			pr1 = unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{
				HeadRepoID: repo.ID,
				Flow:       issues_model.PullRequestFlowAGit,
			})
			if !assert.NotEmpty(t, pr1) {
				return
			}
			assert.Equal(t, 1, pr1.CommitsAhead)
			assert.Equal(t, 0, pr1.CommitsBehind)

			prMsg, err := doAPIGetPullRequest(*ctx, ctx.Username, ctx.Reponame, pr1.Index)(t)
			require.NoError(t, err)

			assert.Equal(t, "user2/"+headBranch, pr1.HeadBranch)
			assert.False(t, prMsg.HasMerged)
			assert.Contains(t, "Testing commit 1", prMsg.Body)
			assert.Equal(t, commit, prMsg.Head.Sha)

			_, _, err = git.NewCommand(git.DefaultContext, "push", "origin").AddDynamicArguments("HEAD:refs/for/master/test/" + headBranch).RunStdString(&git.RunOpts{Dir: dstPath})
			require.NoError(t, err)

			unittest.AssertCount(t, &issues_model.PullRequest{}, pullNum+2)
			pr2 = unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{
				HeadRepoID: repo.ID,
				Index:      pr1.Index + 1,
				Flow:       issues_model.PullRequestFlowAGit,
			})
			if !assert.NotEmpty(t, pr2) {
				return
			}
			assert.Equal(t, 1, pr2.CommitsAhead)
			assert.Equal(t, 0, pr2.CommitsBehind)
			prMsg, err = doAPIGetPullRequest(*ctx, ctx.Username, ctx.Reponame, pr2.Index)(t)
			require.NoError(t, err)

			assert.Equal(t, "user2/test/"+headBranch, pr2.HeadBranch)
			assert.False(t, prMsg.HasMerged)
		})

		if pr1 == nil || pr2 == nil {
			return
		}

		t.Run("AGitLabelIsPresent", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			session := loginUser(t, ctx.Username)

			req := NewRequest(t, "GET", fmt.Sprintf("/%s/%s/pulls/%d", url.PathEscape(ctx.Username), url.PathEscape(ctx.Reponame), pr2.Index))
			resp := session.MakeRequest(t, req, http.StatusOK)
			htmlDoc := NewHTMLParser(t, resp.Body)
			htmlDoc.AssertElement(t, "#agit-label", true)
		})

		t.Run("AddCommit2", func(t *testing.T) {
			err := os.WriteFile(path.Join(dstPath, "test_file"), []byte("## test content \n ## test content 2"), 0o666)
			require.NoError(t, err)

			err = git.AddChanges(dstPath, true)
			require.NoError(t, err)

			err = git.CommitChanges(dstPath, git.CommitChangesOptions{
				Committer: &git.Signature{
					Email: "user2@example.com",
					Name:  "user2",
					When:  time.Now(),
				},
				Author: &git.Signature{
					Email: "user2@example.com",
					Name:  "user2",
					When:  time.Now(),
				},
				Message: "Testing commit 2\n\nLonger description.",
			})
			require.NoError(t, err)
			commit, err = gitRepo.GetRefCommitID("HEAD")
			require.NoError(t, err)
		})

		t.Run("Push2", func(t *testing.T) {
			err := git.NewCommand(git.DefaultContext, "push", "origin", "HEAD:refs/for/master", "-o").AddDynamicArguments("topic=" + headBranch).Run(&git.RunOpts{Dir: dstPath})
			require.NoError(t, err)

			unittest.AssertCount(t, &issues_model.PullRequest{}, pullNum+2)
			prMsg, err := doAPIGetPullRequest(*ctx, ctx.Username, ctx.Reponame, pr1.Index)(t)
			require.NoError(t, err)

			assert.False(t, prMsg.HasMerged)
			assert.Equal(t, commit, prMsg.Head.Sha)

			pr1 = unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{
				HeadRepoID: repo.ID,
				Flow:       issues_model.PullRequestFlowAGit,
				Index:      pr1.Index,
			})
			assert.Equal(t, 2, pr1.CommitsAhead)
			assert.Equal(t, 0, pr1.CommitsBehind)

			_, _, err = git.NewCommand(git.DefaultContext, "push", "origin").AddDynamicArguments("HEAD:refs/for/master/test/" + headBranch).RunStdString(&git.RunOpts{Dir: dstPath})
			require.NoError(t, err)

			unittest.AssertCount(t, &issues_model.PullRequest{}, pullNum+2)
			prMsg, err = doAPIGetPullRequest(*ctx, ctx.Username, ctx.Reponame, pr2.Index)(t)
			require.NoError(t, err)

			assert.False(t, prMsg.HasMerged)
			assert.Equal(t, commit, prMsg.Head.Sha)
		})
		t.Run("PushParams", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			t.Run("NoParams", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				_, _, gitErr := git.NewCommand(git.DefaultContext, "push", "origin").AddDynamicArguments("HEAD:refs/for/master/" + headBranch + "-implicit").RunStdString(&git.RunOpts{Dir: dstPath})
				require.NoError(t, gitErr)

				unittest.AssertCount(t, &issues_model.PullRequest{}, pullNum+3)
				pr3 := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{
					HeadRepoID: repo.ID,
					Flow:       issues_model.PullRequestFlowAGit,
					Index:      pr1.Index + 2,
				})
				assert.NotEmpty(t, pr3)
				err := pr3.LoadIssue(db.DefaultContext)
				require.NoError(t, err)

				_, err2 := doAPIGetPullRequest(*ctx, ctx.Username, ctx.Reponame, pr3.Index)(t)
				require.NoError(t, err2)

				assert.Equal(t, "Testing commit 2", pr3.Issue.Title)
				assert.Contains(t, pr3.Issue.Content, "Longer description.")
			})
			t.Run("TitleOverride", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				_, _, gitErr := git.NewCommand(git.DefaultContext, "push", "origin", "-o", "title=my-shiny-title").AddDynamicArguments("HEAD:refs/for/master/" + headBranch + "-implicit-2").RunStdString(&git.RunOpts{Dir: dstPath})
				require.NoError(t, gitErr)

				unittest.AssertCount(t, &issues_model.PullRequest{}, pullNum+4)
				pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{
					HeadRepoID: repo.ID,
					Flow:       issues_model.PullRequestFlowAGit,
					Index:      pr1.Index + 3,
				})
				assert.NotEmpty(t, pr)
				err := pr.LoadIssue(db.DefaultContext)
				require.NoError(t, err)

				_, err = doAPIGetPullRequest(*ctx, ctx.Username, ctx.Reponame, pr.Index)(t)
				require.NoError(t, err)

				assert.Equal(t, "my-shiny-title", pr.Issue.Title)
				assert.Contains(t, pr.Issue.Content, "Longer description.")
			})

			t.Run("DescriptionOverride", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				_, _, gitErr := git.NewCommand(git.DefaultContext, "push", "origin", "-o", "description=custom").AddDynamicArguments("HEAD:refs/for/master/" + headBranch + "-implicit-3").RunStdString(&git.RunOpts{Dir: dstPath})
				require.NoError(t, gitErr)

				unittest.AssertCount(t, &issues_model.PullRequest{}, pullNum+5)
				pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{
					HeadRepoID: repo.ID,
					Flow:       issues_model.PullRequestFlowAGit,
					Index:      pr1.Index + 4,
				})
				assert.NotEmpty(t, pr)
				err := pr.LoadIssue(db.DefaultContext)
				require.NoError(t, err)

				_, err = doAPIGetPullRequest(*ctx, ctx.Username, ctx.Reponame, pr.Index)(t)
				require.NoError(t, err)

				assert.Equal(t, "Testing commit 2", pr.Issue.Title)
				assert.Contains(t, pr.Issue.Content, "custom")
			})
		})

		upstreamGitRepo, err := git.OpenRepository(git.DefaultContext, filepath.Join(setting.RepoRootPath, ctx.Username, ctx.Reponame+".git"))
		require.NoError(t, err)
		defer upstreamGitRepo.Close()

		t.Run("Force push", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			_, _, gitErr := git.NewCommand(git.DefaultContext, "push", "origin").AddDynamicArguments("HEAD:refs/for/master/" + headBranch + "-force-push").RunStdString(&git.RunOpts{Dir: dstPath})
			require.NoError(t, gitErr)

			unittest.AssertCount(t, &issues_model.PullRequest{}, pullNum+6)
			pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{
				HeadRepoID: repo.ID,
				Flow:       issues_model.PullRequestFlowAGit,
				Index:      pr1.Index + 5,
			})

			headCommitID, err := upstreamGitRepo.GetRefCommitID(pr.GetGitRefName())
			require.NoError(t, err)

			_, _, gitErr = git.NewCommand(git.DefaultContext, "reset", "--hard", "HEAD~1").RunStdString(&git.RunOpts{Dir: dstPath})
			require.NoError(t, gitErr)

			t.Run("Fails", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				_, stdErr, gitErr := git.NewCommand(git.DefaultContext, "push", "origin").AddDynamicArguments("HEAD:refs/for/master/" + headBranch + "-force-push").RunStdString(&git.RunOpts{Dir: dstPath})
				require.Error(t, gitErr)

				assert.Contains(t, stdErr, "-o force-push=true")

				currentHeadCommitID, err := upstreamGitRepo.GetRefCommitID(pr.GetGitRefName())
				require.NoError(t, err)
				assert.Equal(t, headCommitID, currentHeadCommitID)
			})
			t.Run("Succeeds", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				_, _, gitErr := git.NewCommand(git.DefaultContext, "push", "origin", "-o", "force-push").AddDynamicArguments("HEAD:refs/for/master/" + headBranch + "-force-push").RunStdString(&git.RunOpts{Dir: dstPath})
				require.NoError(t, gitErr)

				currentHeadCommitID, err := upstreamGitRepo.GetRefCommitID(pr.GetGitRefName())
				require.NoError(t, err)
				assert.NotEqual(t, headCommitID, currentHeadCommitID)
			})
		})

		t.Run("Branch already contains commit", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			branchCommit, err := upstreamGitRepo.GetBranchCommit("master")
			require.NoError(t, err)

			_, _, gitErr := git.NewCommand(git.DefaultContext, "reset", "--hard").AddDynamicArguments(branchCommit.ID.String() + "~1").RunStdString(&git.RunOpts{Dir: dstPath})
			require.NoError(t, gitErr)

			_, stdErr, gitErr := git.NewCommand(git.DefaultContext, "push", "origin").AddDynamicArguments("HEAD:refs/for/master/" + headBranch + "-already-contains").RunStdString(&git.RunOpts{Dir: dstPath})
			require.Error(t, gitErr)

			assert.Contains(t, stdErr, "already contains this commit")
		})

		t.Run("Merge", doAPIMergePullRequest(*ctx, ctx.Username, ctx.Reponame, pr1.Index))

		t.Run("AGitLabelIsPresent Merged", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			session := loginUser(t, ctx.Username)

			req := NewRequest(t, "GET", fmt.Sprintf("/%s/%s/pulls/%d", url.PathEscape(ctx.Username), url.PathEscape(ctx.Reponame), pr2.Index))
			resp := session.MakeRequest(t, req, http.StatusOK)
			htmlDoc := NewHTMLParser(t, resp.Body)
			htmlDoc.AssertElement(t, "#agit-label", true)
		})

		t.Run("CheckoutMasterAgain", doGitCheckoutBranch(dstPath, "master"))
	}
}

func TestDataAsync_Issue29101(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
		repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})

		resp, err := files_service.ChangeRepoFiles(db.DefaultContext, repo, user, &files_service.ChangeRepoFilesOptions{
			Files: []*files_service.ChangeRepoFile{
				{
					Operation:     "create",
					TreePath:      "test.txt",
					ContentReader: bytes.NewReader(make([]byte, 10000)),
				},
			},
			OldBranch: repo.DefaultBranch,
			NewBranch: repo.DefaultBranch,
		})
		require.NoError(t, err)

		sha := resp.Commit.SHA

		gitRepo, err := gitrepo.OpenRepository(db.DefaultContext, repo)
		require.NoError(t, err)
		defer gitRepo.Close()

		commit, err := gitRepo.GetCommit(sha)
		require.NoError(t, err)

		entry, err := commit.GetTreeEntryByPath("test.txt")
		require.NoError(t, err)

		b := entry.Blob()

		r, err := b.DataAsync()
		require.NoError(t, err)
		defer r.Close()

		r2, err := b.DataAsync()
		require.NoError(t, err)
		defer r2.Close()
	})
}

func doLFSNoAccess(ctx APITestContext, publicKeyID int64, objectFormat git.ObjectFormat) func(*testing.T) {
	return func(t *testing.T) {
		// This is set in withKeyFile
		sshCommand := os.Getenv("GIT_SSH_COMMAND")

		// Sanity check, because we are going to execute whatever is in here.
		require.True(t, strings.HasPrefix(sshCommand, "ssh "))

		// We really have to split on the arguments and pass them individually.
		sshOptions, err := shellquote.Split(strings.TrimPrefix(sshCommand, "ssh "))
		require.NoError(t, err)

		sshOptions = append(sshOptions, "-p "+strconv.Itoa(setting.SSH.ListenPort), "git@"+setting.SSH.ListenHost)

		cmd := exec.CommandContext(t.Context(), "ssh", append(sshOptions, "git-lfs-authenticate", "user40/repo60.git", "upload")...)
		stderr := bytes.Buffer{}
		cmd.Stderr = &stderr

		require.ErrorContains(t, cmd.Run(), "exit status 1")
		if objectFormat.Name() == "sha1" {
			assert.Contains(t, stderr.String(), fmt.Sprintf("Forgejo: User: 2:user2 with Key: %d:test-key-sha1 is not authorized to write to user40/repo60.", publicKeyID))
		} else {
			assert.Contains(t, stderr.String(), fmt.Sprintf("Forgejo: User: 2:user2 with Key: %d:test-key-sha256 is not authorized to write to user40/repo60.", publicKeyID))
		}
	}
}
