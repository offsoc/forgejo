// Copyright Earl Warren <contact@earl-warren.org>
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"net/url"
	"os"
	"path"
	"testing"
	"time"

	actions_model "forgejo.org/models/actions"
	auth_model "forgejo.org/models/auth"
	"forgejo.org/models/db"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/git"
	api "forgejo.org/modules/structs"
	"forgejo.org/tests"

	"github.com/stretchr/testify/require"
)

func TestActionsUserGit(t *testing.T) {
	onGiteaRun(t, testActionsUserGit)
}

func NewActionsUserTestContext(t *testing.T, username, reponame string) APITestContext {
	t.Helper()

	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{Name: reponame})
	repoOwner := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: username})

	task := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionTask{ID: 47})
	task.RepoID = repo.ID
	task.OwnerID = repoOwner.ID
	task.GenerateToken()

	actions_model.UpdateTask(db.DefaultContext, task)
	return APITestContext{
		Session:  emptyTestSession(t),
		Token:    task.Token,
		Username: username,
		Reponame: reponame,
	}
}

func testActionsUserGit(t *testing.T, u *url.URL) {
	username := "user2"
	reponame := "repo1"
	httpContext := NewAPITestContext(t, username, reponame, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)

	for _, testCase := range []struct {
		name string
		head string
		ctx  APITestContext
	}{
		{
			name: "UserTypeIndividual",
			head: "individualhead",
			ctx:  httpContext,
		},
		{
			name: "ActionsUser",
			head: "actionsuserhead",
			ctx:  NewActionsUserTestContext(t, username, reponame),
		},
	} {
		t.Run("CreatePR "+testCase.name, func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			dstPath := t.TempDir()
			u.Path = httpContext.GitPath()
			u.User = url.UserPassword(httpContext.Username, userPassword)
			t.Run("Clone", doGitClone(dstPath, u))
			t.Run("PopulateBranch", doActionsUserPopulateBranch(dstPath, &httpContext, "master", testCase.head))
			t.Run("CreatePR", doActionsUserPR(httpContext, testCase.ctx, "master", testCase.head))
		})
	}
}

func doActionsUserPopulateBranch(dstPath string, ctx *APITestContext, baseBranch, headBranch string) func(t *testing.T) {
	return func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

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
		})

		t.Run("Push", func(t *testing.T) {
			err := git.NewCommand(git.DefaultContext, "push", "origin").AddDynamicArguments("HEAD:refs/heads/" + headBranch).Run(&git.RunOpts{Dir: dstPath})
			require.NoError(t, err)
		})
	}
}

func doActionsUserPR(ctx, doerCtx APITestContext, baseBranch, headBranch string) func(t *testing.T) {
	return func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		var pr api.PullRequest
		var err error

		// Create a test pullrequest
		t.Run("CreatePullRequest", func(t *testing.T) {
			pr, err = doAPICreatePullRequest(doerCtx, ctx.Username, ctx.Reponame, baseBranch, headBranch)(t)
			require.NoError(t, err)
		})
		doerCtx.ExpectedCode = http.StatusCreated
		t.Run("AutoMergePR", doAPIAutoMergePullRequest(doerCtx, ctx.Username, ctx.Reponame, pr.Index))
		// Ensure the PR page works
		t.Run("EnsureCanSeePull", doEnsureCanSeePull(ctx, pr, true))
	}
}
