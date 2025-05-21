// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"forgejo.org/models"
	auth_model "forgejo.org/models/auth"
	"forgejo.org/models/db"
	git_model "forgejo.org/models/git"
	issues_model "forgejo.org/models/issues"
	pull_model "forgejo.org/models/pull"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/models/webhook"
	"forgejo.org/modules/git"
	"forgejo.org/modules/gitrepo"
	"forgejo.org/modules/hostmatcher"
	"forgejo.org/modules/queue"
	"forgejo.org/modules/setting"
	api "forgejo.org/modules/structs"
	"forgejo.org/modules/test"
	"forgejo.org/modules/translation"
	"forgejo.org/services/automerge"
	forgejo_context "forgejo.org/services/context"
	"forgejo.org/services/forms"
	"forgejo.org/services/pull"
	commitstatus_service "forgejo.org/services/repository/commitstatus"
	webhook_service "forgejo.org/services/webhook"
	"forgejo.org/tests"

	"github.com/hashicorp/go-version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type optionsPullMerge map[string]string

func testPullMerge(t *testing.T, session *TestSession, user, repo, pullnum string, mergeStyle repo_model.MergeStyle, deleteBranch bool) *httptest.ResponseRecorder {
	options := optionsPullMerge{
		"do": string(mergeStyle),
	}
	if deleteBranch {
		options["delete_branch_after_merge"] = "on"
	}

	return testPullMergeForm(t, session, http.StatusOK, user, repo, pullnum, options)
}

func testPullMergeForm(t *testing.T, session *TestSession, expectedCode int, user, repo, pullnum string, addOptions optionsPullMerge) *httptest.ResponseRecorder {
	req := NewRequest(t, "GET", path.Join(user, repo, "pulls", pullnum))
	resp := session.MakeRequest(t, req, http.StatusOK)

	htmlDoc := NewHTMLParser(t, resp.Body)
	link := path.Join(user, repo, "pulls", pullnum, "merge")

	options := map[string]string{
		"_csrf": htmlDoc.GetCSRF(),
	}
	for k, v := range addOptions {
		options[k] = v
	}

	req = NewRequestWithValues(t, "POST", link, options)
	resp = session.MakeRequest(t, req, expectedCode)

	if expectedCode == http.StatusOK {
		respJSON := struct {
			Redirect string
		}{}
		DecodeJSON(t, resp, &respJSON)

		assert.Equal(t, fmt.Sprintf("/%s/%s/pulls/%s", user, repo, pullnum), respJSON.Redirect)
	}

	return resp
}

func testPullCleanUp(t *testing.T, session *TestSession, user, repo, pullnum string) *httptest.ResponseRecorder {
	req := NewRequest(t, "GET", path.Join(user, repo, "pulls", pullnum))
	resp := session.MakeRequest(t, req, http.StatusOK)

	// Click the little button to create a pull
	htmlDoc := NewHTMLParser(t, resp.Body)
	link, exists := htmlDoc.doc.Find(".timeline-item .delete-button").Attr("data-url")
	assert.True(t, exists, "The template has changed, can not find delete button url")
	req = NewRequestWithValues(t, "POST", link, map[string]string{
		"_csrf": htmlDoc.GetCSRF(),
	})
	resp = session.MakeRequest(t, req, http.StatusOK)

	return resp
}

// returns the hook tasks, order by ID desc.
func retrieveHookTasks(t *testing.T, hookID int64, activateWebhook bool) []*webhook.HookTask {
	t.Helper()
	if activateWebhook {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}))
		t.Cleanup(s.Close)
		updated, err := db.GetEngine(db.DefaultContext).ID(hookID).Cols("is_active", "url").Update(webhook.Webhook{
			IsActive: true,
			URL:      s.URL,
		})

		// allow webhook deliveries on localhost
		t.Cleanup(test.MockVariableValue(&setting.Webhook.AllowedHostList, hostmatcher.MatchBuiltinLoopback))
		webhook_service.Init()

		assert.Equal(t, int64(1), updated)
		require.NoError(t, err)
	}

	hookTasks, err := webhook.HookTasks(db.DefaultContext, hookID, 1)
	require.NoError(t, err)
	return hookTasks
}

func TestPullMerge(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, giteaURL *url.URL) {
		hookTasks := retrieveHookTasks(t, 1, true)
		hookTasksLenBefore := len(hookTasks)

		session := loginUser(t, "user1")
		testRepoFork(t, session, "user2", "repo1", "user1", "repo1")
		testEditFile(t, session, "user1", "repo1", "master", "README.md", "Hello, World (Edited)\n")

		resp := testPullCreate(t, session, "user1", "repo1", false, "master", "master", "This is a pull title")

		elem := strings.Split(test.RedirectURL(resp), "/")
		assert.Equal(t, "pulls", elem[3])
		testPullMerge(t, session, elem[1], elem[2], elem[4], repo_model.MergeStyleMerge, false)

		hookTasks = retrieveHookTasks(t, 1, false)
		assert.Len(t, hookTasks, hookTasksLenBefore+1)
	})
}

func TestPullRebase(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, giteaURL *url.URL) {
		hookTasks := retrieveHookTasks(t, 1, true)
		hookTasksLenBefore := len(hookTasks)

		session := loginUser(t, "user1")
		testRepoFork(t, session, "user2", "repo1", "user1", "repo1")
		testEditFile(t, session, "user1", "repo1", "master", "README.md", "Hello, World (Edited)\n")

		resp := testPullCreate(t, session, "user1", "repo1", false, "master", "master", "This is a pull title")

		elem := strings.Split(test.RedirectURL(resp), "/")
		assert.Equal(t, "pulls", elem[3])
		testPullMerge(t, session, elem[1], elem[2], elem[4], repo_model.MergeStyleRebase, false)

		hookTasks = retrieveHookTasks(t, 1, false)
		assert.Len(t, hookTasks, hookTasksLenBefore+1)
	})
}

func TestPullRebaseMerge(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, giteaURL *url.URL) {
		hookTasks := retrieveHookTasks(t, 1, true)
		hookTasksLenBefore := len(hookTasks)

		session := loginUser(t, "user1")
		testRepoFork(t, session, "user2", "repo1", "user1", "repo1")
		testEditFile(t, session, "user1", "repo1", "master", "README.md", "Hello, World (Edited)\n")

		resp := testPullCreate(t, session, "user1", "repo1", false, "master", "master", "This is a pull title")

		elem := strings.Split(test.RedirectURL(resp), "/")
		assert.Equal(t, "pulls", elem[3])
		testPullMerge(t, session, elem[1], elem[2], elem[4], repo_model.MergeStyleRebaseMerge, false)

		hookTasks = retrieveHookTasks(t, 1, false)
		assert.Len(t, hookTasks, hookTasksLenBefore+1)
	})
}

func TestPullSquash(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, giteaURL *url.URL) {
		hookTasks := retrieveHookTasks(t, 1, true)
		hookTasksLenBefore := len(hookTasks)

		session := loginUser(t, "user1")
		testRepoFork(t, session, "user2", "repo1", "user1", "repo1")
		testEditFile(t, session, "user1", "repo1", "master", "README.md", "Hello, World (Edited)\n")
		testEditFile(t, session, "user1", "repo1", "master", "README.md", "Hello, World (Edited!)\n")

		resp := testPullCreate(t, session, "user1", "repo1", false, "master", "master", "This is a pull title")

		elem := strings.Split(test.RedirectURL(resp), "/")
		assert.Equal(t, "pulls", elem[3])
		testPullMerge(t, session, elem[1], elem[2], elem[4], repo_model.MergeStyleSquash, false)

		hookTasks = retrieveHookTasks(t, 1, false)
		assert.Len(t, hookTasks, hookTasksLenBefore+1)
	})
}

func TestPullCleanUpAfterMerge(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, giteaURL *url.URL) {
		session := loginUser(t, "user1")
		testRepoFork(t, session, "user2", "repo1", "user1", "repo1")
		testEditFileToNewBranch(t, session, "user1", "repo1", "master", "feature/test", "README.md", "Hello, World (Edited - TestPullCleanUpAfterMerge)\n")

		resp := testPullCreate(t, session, "user1", "repo1", false, "master", "feature/test", "This is a pull title")

		elem := strings.Split(test.RedirectURL(resp), "/")
		assert.Equal(t, "pulls", elem[3])
		testPullMerge(t, session, elem[1], elem[2], elem[4], repo_model.MergeStyleMerge, false)

		// Check PR branch deletion
		resp = testPullCleanUp(t, session, elem[1], elem[2], elem[4])
		respJSON := struct {
			Redirect string
		}{}
		DecodeJSON(t, resp, &respJSON)

		assert.NotEmpty(t, respJSON.Redirect, "Redirected URL is not found")

		elem = strings.Split(respJSON.Redirect, "/")
		assert.Equal(t, "pulls", elem[3])

		// Check branch deletion result
		req := NewRequest(t, "GET", respJSON.Redirect)
		resp = session.MakeRequest(t, req, http.StatusOK)

		htmlDoc := NewHTMLParser(t, resp.Body)
		resultMsg := htmlDoc.doc.Find(".ui.message>p").Text()

		assert.Equal(t, "Branch \"user1/repo1:feature/test\" has been deleted.", resultMsg)
	})
}

func TestCantMergeWorkInProgress(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, giteaURL *url.URL) {
		session := loginUser(t, "user1")
		testRepoFork(t, session, "user2", "repo1", "user1", "repo1")
		testEditFile(t, session, "user1", "repo1", "master", "README.md", "Hello, World (Edited)\n")

		resp := testPullCreate(t, session, "user1", "repo1", false, "master", "master", "[wip] This is a pull title")

		req := NewRequest(t, "GET", test.RedirectURL(resp))
		resp = session.MakeRequest(t, req, http.StatusOK)
		htmlDoc := NewHTMLParser(t, resp.Body)
		text := strings.TrimSpace(htmlDoc.doc.Find(".merge-section > .item").Last().Text())
		assert.NotEmpty(t, text, "Can't find WIP text")

		assert.Contains(t, text, translation.NewLocale("en-US").TrString("repo.pulls.cannot_merge_work_in_progress"), "Unable to find WIP text")
		assert.Contains(t, text, "[wip]", "Unable to find WIP text")
	})
}

func TestCantMergeConflict(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, giteaURL *url.URL) {
		session := loginUser(t, "user1")
		testRepoFork(t, session, "user2", "repo1", "user1", "repo1")
		testEditFileToNewBranch(t, session, "user1", "repo1", "master", "conflict", "README.md", "Hello, World (Edited Once)\n")
		testEditFileToNewBranch(t, session, "user1", "repo1", "master", "base", "README.md", "Hello, World (Edited Twice)\n")

		// Use API to create a conflicting pr
		token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository)
		req := NewRequestWithJSON(t, http.MethodPost, fmt.Sprintf("/api/v1/repos/%s/%s/pulls", "user1", "repo1"), &api.CreatePullRequestOption{
			Head:  "conflict",
			Base:  "base",
			Title: "create a conflicting pr",
		}).AddTokenAuth(token)
		session.MakeRequest(t, req, http.StatusCreated)

		// Now this PR will be marked conflict - or at least a race will do - so drop down to pure code at this point...
		user1 := unittest.AssertExistsAndLoadBean(t, &user_model.User{
			Name: "user1",
		})
		repo1 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{
			OwnerID: user1.ID,
			Name:    "repo1",
		})

		pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{
			HeadRepoID: repo1.ID,
			BaseRepoID: repo1.ID,
			HeadBranch: "conflict",
			BaseBranch: "base",
		})

		gitRepo, err := gitrepo.OpenRepository(git.DefaultContext, repo1)
		require.NoError(t, err)

		t.Run("Rebase", func(t *testing.T) {
			t.Run("Git version without replay", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()
				oldVersion, err := version.NewVersion("2.43.0")
				require.NoError(t, err)
				defer test.MockVariableValue(&git.GitVersion, oldVersion)()

				err = pull.Merge(t.Context(), pr, user1, gitRepo, repo_model.MergeStyleRebase, "", "CONFLICT", false)
				require.Error(t, err, "Merge should return an error due to conflict")
				assert.True(t, models.IsErrRebaseConflicts(err), "Merge error is not a conflict error")
			})
			t.Run("Git version with replay", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()
				if git.CheckGitVersionAtLeast("2.44") != nil {
					t.SkipNow()
				}

				err = pull.Merge(t.Context(), pr, user1, gitRepo, repo_model.MergeStyleRebase, "", "CONFLICT", false)
				require.Error(t, err, "Merge should return an error due to conflict")
				assert.True(t, models.IsErrRebaseConflicts(err), "Merge error is not a conflict error")
			})
		})

		t.Run("Merge", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			err = pull.Merge(t.Context(), pr, user1, gitRepo, repo_model.MergeStyleMerge, "", "CONFLICT", false)
			require.Error(t, err, "Merge should return an error due to conflict")
			assert.True(t, models.IsErrMergeConflicts(err), "Merge error is not a conflict error")
		})

		gitRepo.Close()
	})
}

func TestCantMergeUnrelated(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, giteaURL *url.URL) {
		session := loginUser(t, "user1")
		testRepoFork(t, session, "user2", "repo1", "user1", "repo1")
		testEditFileToNewBranch(t, session, "user1", "repo1", "master", "base", "README.md", "Hello, World (Edited Twice)\n")

		// Now we want to create a commit on a branch that is totally unrelated to our current head
		// Drop down to pure code at this point
		user1 := unittest.AssertExistsAndLoadBean(t, &user_model.User{
			Name: "user1",
		})
		repo1 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{
			OwnerID: user1.ID,
			Name:    "repo1",
		})
		path := repo_model.RepoPath(user1.Name, repo1.Name)

		err := git.NewCommand(git.DefaultContext, "read-tree", "--empty").Run(&git.RunOpts{Dir: path})
		require.NoError(t, err)

		stdin := bytes.NewBufferString("Unrelated File")
		var stdout strings.Builder
		err = git.NewCommand(git.DefaultContext, "hash-object", "-w", "--stdin").Run(&git.RunOpts{
			Dir:    path,
			Stdin:  stdin,
			Stdout: &stdout,
		})

		require.NoError(t, err)
		sha := strings.TrimSpace(stdout.String())

		_, _, err = git.NewCommand(git.DefaultContext, "update-index", "--add", "--replace", "--cacheinfo").AddDynamicArguments("100644", sha, "somewher-over-the-rainbow").RunStdString(&git.RunOpts{Dir: path})
		require.NoError(t, err)

		treeSha, _, err := git.NewCommand(git.DefaultContext, "write-tree").RunStdString(&git.RunOpts{Dir: path})
		require.NoError(t, err)
		treeSha = strings.TrimSpace(treeSha)

		commitTimeStr := time.Now().Format(time.RFC3339)
		doerSig := user1.NewGitSig()
		env := append(os.Environ(),
			"GIT_AUTHOR_NAME="+doerSig.Name,
			"GIT_AUTHOR_EMAIL="+doerSig.Email,
			"GIT_AUTHOR_DATE="+commitTimeStr,
			"GIT_COMMITTER_NAME="+doerSig.Name,
			"GIT_COMMITTER_EMAIL="+doerSig.Email,
			"GIT_COMMITTER_DATE="+commitTimeStr,
		)

		messageBytes := new(bytes.Buffer)
		_, _ = messageBytes.WriteString("Unrelated")
		_, _ = messageBytes.WriteString("\n")

		stdout.Reset()
		err = git.NewCommand(git.DefaultContext, "commit-tree").AddDynamicArguments(treeSha).
			Run(&git.RunOpts{
				Env:    env,
				Dir:    path,
				Stdin:  messageBytes,
				Stdout: &stdout,
			})
		require.NoError(t, err)
		commitSha := strings.TrimSpace(stdout.String())

		_, _, err = git.NewCommand(git.DefaultContext, "branch", "unrelated").AddDynamicArguments(commitSha).RunStdString(&git.RunOpts{Dir: path})
		require.NoError(t, err)

		testEditFileToNewBranch(t, session, "user1", "repo1", "master", "conflict", "README.md", "Hello, World (Edited Once)\n")

		// Use API to create a conflicting pr
		token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository)
		req := NewRequestWithJSON(t, http.MethodPost, fmt.Sprintf("/api/v1/repos/%s/%s/pulls", "user1", "repo1"), &api.CreatePullRequestOption{
			Head:  "unrelated",
			Base:  "base",
			Title: "create an unrelated pr",
		}).AddTokenAuth(token)
		session.MakeRequest(t, req, http.StatusCreated)

		// Now this PR could be marked conflict - or at least a race may occur - so drop down to pure code at this point...
		gitRepo, err := gitrepo.OpenRepository(git.DefaultContext, repo1)
		require.NoError(t, err)
		pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{
			HeadRepoID: repo1.ID,
			BaseRepoID: repo1.ID,
			HeadBranch: "unrelated",
			BaseBranch: "base",
		})

		err = pull.Merge(t.Context(), pr, user1, gitRepo, repo_model.MergeStyleMerge, "", "UNRELATED", false)
		require.Error(t, err, "Merge should return an error due to unrelated")
		assert.True(t, models.IsErrMergeUnrelatedHistories(err), "Merge error is not a unrelated histories error")
		gitRepo.Close()
	})
}

func TestFastForwardOnlyMerge(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, giteaURL *url.URL) {
		session := loginUser(t, "user1")
		testRepoFork(t, session, "user2", "repo1", "user1", "repo1")
		testEditFileToNewBranch(t, session, "user1", "repo1", "master", "update", "README.md", "Hello, World 2\n")

		// Use API to create a pr from update to master
		token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository)
		req := NewRequestWithJSON(t, http.MethodPost, fmt.Sprintf("/api/v1/repos/%s/%s/pulls", "user1", "repo1"), &api.CreatePullRequestOption{
			Head:  "update",
			Base:  "master",
			Title: "create a pr that can be fast-forward-only merged",
		}).AddTokenAuth(token)
		session.MakeRequest(t, req, http.StatusCreated)

		user1 := unittest.AssertExistsAndLoadBean(t, &user_model.User{
			Name: "user1",
		})
		repo1 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{
			OwnerID: user1.ID,
			Name:    "repo1",
		})

		pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{
			HeadRepoID: repo1.ID,
			BaseRepoID: repo1.ID,
			HeadBranch: "update",
			BaseBranch: "master",
		})

		gitRepo, err := git.OpenRepository(git.DefaultContext, repo_model.RepoPath(user1.Name, repo1.Name))
		require.NoError(t, err)

		err = pull.Merge(t.Context(), pr, user1, gitRepo, repo_model.MergeStyleFastForwardOnly, "", "FAST-FORWARD-ONLY", false)

		require.NoError(t, err)

		gitRepo.Close()
	})
}

func TestCantFastForwardOnlyMergeDiverging(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, giteaURL *url.URL) {
		session := loginUser(t, "user1")
		testRepoFork(t, session, "user2", "repo1", "user1", "repo1")
		testEditFileToNewBranch(t, session, "user1", "repo1", "master", "diverging", "README.md", "Hello, World diverged\n")
		testEditFile(t, session, "user1", "repo1", "master", "README.md", "Hello, World 2\n")

		// Use API to create a pr from diverging to update
		token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository)
		req := NewRequestWithJSON(t, http.MethodPost, fmt.Sprintf("/api/v1/repos/%s/%s/pulls", "user1", "repo1"), &api.CreatePullRequestOption{
			Head:  "diverging",
			Base:  "master",
			Title: "create a pr from a diverging branch",
		}).AddTokenAuth(token)
		session.MakeRequest(t, req, http.StatusCreated)

		user1 := unittest.AssertExistsAndLoadBean(t, &user_model.User{
			Name: "user1",
		})
		repo1 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{
			OwnerID: user1.ID,
			Name:    "repo1",
		})

		pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{
			HeadRepoID: repo1.ID,
			BaseRepoID: repo1.ID,
			HeadBranch: "diverging",
			BaseBranch: "master",
		})

		gitRepo, err := git.OpenRepository(git.DefaultContext, repo_model.RepoPath(user1.Name, repo1.Name))
		require.NoError(t, err)

		err = pull.Merge(t.Context(), pr, user1, gitRepo, repo_model.MergeStyleFastForwardOnly, "", "DIVERGING", false)

		require.Error(t, err, "Merge should return an error due to being for a diverging branch")
		assert.True(t, models.IsErrMergeDivergingFastForwardOnly(err), "Merge error is not a diverging fast-forward-only error")

		gitRepo.Close()
	})
}

func TestPullRetargetChildOnBranchDelete(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, giteaURL *url.URL) {
		session := loginUser(t, "user1")
		testEditFileToNewBranch(t, session, "user2", "repo1", "master", "base-pr", "README.md", "Hello, World\n(Edited - TestPullRetargetOnCleanup - base PR)\n")
		testRepoFork(t, session, "user2", "repo1", "user1", "repo1")
		testEditFileToNewBranch(t, session, "user1", "repo1", "base-pr", "child-pr", "README.md", "Hello, World\n(Edited - TestPullRetargetOnCleanup - base PR)\n(Edited - TestPullRetargetOnCleanup - child PR)")

		respBasePR := testPullCreate(t, session, "user2", "repo1", true, "master", "base-pr", "Base Pull Request")
		elemBasePR := strings.Split(test.RedirectURL(respBasePR), "/")
		assert.Equal(t, "pulls", elemBasePR[3])

		respChildPR := testPullCreate(t, session, "user1", "repo1", false, "base-pr", "child-pr", "Child Pull Request")
		elemChildPR := strings.Split(test.RedirectURL(respChildPR), "/")
		assert.Equal(t, "pulls", elemChildPR[3])

		testPullMerge(t, session, elemBasePR[1], elemBasePR[2], elemBasePR[4], repo_model.MergeStyleMerge, true)

		// Check child PR
		req := NewRequest(t, "GET", test.RedirectURL(respChildPR))
		resp := session.MakeRequest(t, req, http.StatusOK)

		htmlDoc := NewHTMLParser(t, resp.Body)
		targetBranch := htmlDoc.doc.Find("#branch_target>a").Text()
		prStatus := strings.TrimSpace(htmlDoc.doc.Find(".issue-title-meta>.issue-state-label").Text())

		assert.Equal(t, "master", targetBranch)
		assert.Equal(t, "Open", prStatus)
	})
}

func TestPullDontRetargetChildOnWrongRepo(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, giteaURL *url.URL) {
		session := loginUser(t, "user1")
		testRepoFork(t, session, "user2", "repo1", "user1", "repo1")
		testEditFileToNewBranch(t, session, "user1", "repo1", "master", "base-pr", "README.md", "Hello, World\n(Edited - TestPullDontRetargetChildOnWrongRepo - base PR)\n")
		testEditFileToNewBranch(t, session, "user1", "repo1", "base-pr", "child-pr", "README.md", "Hello, World\n(Edited - TestPullDontRetargetChildOnWrongRepo - base PR)\n(Edited - TestPullDontRetargetChildOnWrongRepo - child PR)")

		respBasePR := testPullCreate(t, session, "user1", "repo1", false, "master", "base-pr", "Base Pull Request")
		elemBasePR := strings.Split(test.RedirectURL(respBasePR), "/")
		assert.Equal(t, "pulls", elemBasePR[3])

		respChildPR := testPullCreate(t, session, "user1", "repo1", true, "base-pr", "child-pr", "Child Pull Request")
		elemChildPR := strings.Split(test.RedirectURL(respChildPR), "/")
		assert.Equal(t, "pulls", elemChildPR[3])

		testPullMerge(t, session, elemBasePR[1], elemBasePR[2], elemBasePR[4], repo_model.MergeStyleMerge, true)

		// Check child PR
		req := NewRequest(t, "GET", test.RedirectURL(respChildPR))
		resp := session.MakeRequest(t, req, http.StatusOK)

		htmlDoc := NewHTMLParser(t, resp.Body)
		targetBranch := htmlDoc.doc.Find("#branch_target>a").Text()
		prStatus := strings.TrimSpace(htmlDoc.doc.Find(".issue-title-meta>.issue-state-label").Text())

		assert.Equal(t, "base-pr", targetBranch)
		assert.Equal(t, "Closed", prStatus)
	})
}

func TestPullMergeIndexerNotifier(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, giteaURL *url.URL) {
		// create a pull request
		session := loginUser(t, "user1")
		testRepoFork(t, session, "user2", "repo1", "user1", "repo1")
		testEditFile(t, session, "user1", "repo1", "master", "README.md", "Hello, World (Edited)\n")
		createPullResp := testPullCreate(t, session, "user1", "repo1", false, "master", "master", "Indexer notifier test pull")

		require.NoError(t, queue.GetManager().FlushAll(t.Context(), 0))

		repo1 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{
			OwnerName: "user2",
			Name:      "repo1",
		})
		issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{
			RepoID:   repo1.ID,
			Title:    "Indexer notifier test pull",
			IsPull:   true,
			IsClosed: false,
		})

		// build the request for searching issues
		link, _ := url.Parse("/api/v1/repos/issues/search")
		query := url.Values{}
		query.Add("state", "closed")
		query.Add("type", "pulls")
		query.Add("q", "notifier")
		link.RawQuery = query.Encode()

		// search issues
		searchIssuesResp := session.MakeRequest(t, NewRequest(t, "GET", link.String()), http.StatusOK)
		var apiIssuesBefore []*api.Issue
		DecodeJSON(t, searchIssuesResp, &apiIssuesBefore)
		assert.Empty(t, apiIssuesBefore)

		// merge the pull request
		elem := strings.Split(test.RedirectURL(createPullResp), "/")
		assert.Equal(t, "pulls", elem[3])
		testPullMerge(t, session, elem[1], elem[2], elem[4], repo_model.MergeStyleMerge, false)

		// check if the issue is closed
		issue = unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{
			ID: issue.ID,
		})
		assert.True(t, issue.IsClosed)

		require.NoError(t, queue.GetManager().FlushAll(t.Context(), 0))

		// search issues again
		searchIssuesResp = session.MakeRequest(t, NewRequest(t, "GET", link.String()), http.StatusOK)
		var apiIssuesAfter []*api.Issue
		DecodeJSON(t, searchIssuesResp, &apiIssuesAfter)
		if assert.Len(t, apiIssuesAfter, 1) {
			assert.Equal(t, issue.ID, apiIssuesAfter[0].ID)
		}
	})
}

func testResetRepo(t *testing.T, repoPath, branch, commitID string) {
	f, err := os.OpenFile(filepath.Join(repoPath, "refs", "heads", branch), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	require.NoError(t, err)
	_, err = f.WriteString(commitID + "\n")
	require.NoError(t, err)
	f.Close()

	repo, err := git.OpenRepository(t.Context(), repoPath)
	require.NoError(t, err)
	defer repo.Close()
	id, err := repo.GetBranchCommitID(branch)
	require.NoError(t, err)
	assert.Equal(t, commitID, id)
}

func TestPullMergeBranchProtect(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		admin := "user1"
		owner := "user5"
		notOwner := "user4"
		repo := "repo4"

		dstPath := t.TempDir()

		u.Path = fmt.Sprintf("%s/%s.git", owner, repo)
		u.User = url.UserPassword(owner, userPassword)

		t.Run("Clone", doGitClone(dstPath, u))

		for _, testCase := range []struct {
			name          string
			doer          string
			expectedCode  map[string]int
			filename      string
			protectBranch parameterProtectBranch
		}{
			{
				name:         "SuccessAdminNotEnoughMergeRequiredApprovals",
				doer:         admin,
				expectedCode: map[string]int{"api": http.StatusOK, "web": http.StatusOK},
				filename:     "branch-data-file-",
				protectBranch: parameterProtectBranch{
					"required_approvals": "1",
					"apply_to_admins":    "true",
				},
			},
			{
				name:         "FailOwnerProtectedFile",
				doer:         owner,
				expectedCode: map[string]int{"api": http.StatusMethodNotAllowed, "web": http.StatusBadRequest},
				filename:     "protected-file-",
				protectBranch: parameterProtectBranch{
					"protected_file_patterns": "protected-file-*",
					"apply_to_admins":         "true",
				},
			},
			{
				name:         "OwnerProtectedFile",
				doer:         owner,
				expectedCode: map[string]int{"api": http.StatusOK, "web": http.StatusOK},
				filename:     "protected-file-",
				protectBranch: parameterProtectBranch{
					"protected_file_patterns": "protected-file-*",
					"apply_to_admins":         "false",
				},
			},
			{
				name:         "FailNotOwnerProtectedFile",
				doer:         notOwner,
				expectedCode: map[string]int{"api": http.StatusMethodNotAllowed, "web": http.StatusBadRequest},
				filename:     "protected-file-",
				protectBranch: parameterProtectBranch{
					"protected_file_patterns": "protected-file-*",
				},
			},
			{
				name:         "FailOwnerNotEnoughMergeRequiredApprovals",
				doer:         owner,
				expectedCode: map[string]int{"api": http.StatusMethodNotAllowed, "web": http.StatusBadRequest},
				filename:     "branch-data-file-",
				protectBranch: parameterProtectBranch{
					"required_approvals": "1",
					"apply_to_admins":    "true",
				},
			},
			{
				name:         "SuccessOwnerNotEnoughMergeRequiredApprovals",
				doer:         owner,
				expectedCode: map[string]int{"api": http.StatusOK, "web": http.StatusOK},
				filename:     "branch-data-file-",
				protectBranch: parameterProtectBranch{
					"required_approvals": "1",
					"apply_to_admins":    "false",
				},
			},
			{
				name:         "FailNotOwnerNotEnoughMergeRequiredApprovals",
				doer:         notOwner,
				expectedCode: map[string]int{"api": http.StatusMethodNotAllowed, "web": http.StatusBadRequest},
				filename:     "branch-data-file-",
				protectBranch: parameterProtectBranch{
					"required_approvals": "1",
					"apply_to_admins":    "false",
				},
			},
			{
				name:         "SuccessNotOwner",
				doer:         notOwner,
				expectedCode: map[string]int{"api": http.StatusOK, "web": http.StatusOK},
				filename:     "branch-data-file-",
				protectBranch: parameterProtectBranch{
					"required_approvals": "0",
				},
			},
		} {
			mergeWith := func(t *testing.T, ctx APITestContext, apiOrWeb string, expectedCode int, pr int64) {
				switch apiOrWeb {
				case "api":
					ctx.ExpectedCode = expectedCode
					doAPIMergePullRequestForm(t, ctx, owner, repo, pr,
						&forms.MergePullRequestForm{
							MergeMessageField: "doAPIMergePullRequest Merge",
							Do:                string(repo_model.MergeStyleMerge),
							ForceMerge:        true,
						})
					ctx.ExpectedCode = 0
				case "web":
					testPullMergeForm(t, ctx.Session, expectedCode, owner, repo, fmt.Sprintf("%d", pr), optionsPullMerge{
						"do":          string(repo_model.MergeStyleMerge),
						"force_merge": "true",
					})
				default:
					panic(apiOrWeb)
				}
			}
			for _, withAPIOrWeb := range []string{"api", "web"} {
				t.Run(testCase.name+" "+withAPIOrWeb, func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()
					branch := testCase.name + "-" + withAPIOrWeb
					unprotected := branch + "-unprotected"
					doGitCheckoutBranch(dstPath, "master")(t)
					doGitCreateBranch(dstPath, branch)(t)
					doGitPushTestRepository(dstPath, "origin", branch)(t)

					ctx := NewAPITestContext(t, owner, repo, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)
					doProtectBranch(ctx, branch, testCase.protectBranch)(t)

					ctx = NewAPITestContext(t, testCase.doer, "not used", auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)
					ctx.Username = owner
					ctx.Reponame = repo
					generateCommitWithNewData(t, littleSize, dstPath, "user2@example.com", "User Two", testCase.filename)
					doGitPushTestRepository(dstPath, "origin", branch+":"+unprotected)(t)
					pr, err := doAPICreatePullRequest(ctx, owner, repo, branch, unprotected)(t)
					require.NoError(t, err)
					mergeWith(t, ctx, withAPIOrWeb, testCase.expectedCode[withAPIOrWeb], pr.Index)
				})
			}
		}
	})
}

func testPullAutoMergeAfterCommitStatusSucceed(t *testing.T, ctx APITestContext, forkName string, approval, deleteBranch, withAPI bool) {
	// prepare environment (fork repo, create user)
	user1 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
	testRepoFork(t, ctx.Session, "user2", "repo1", "user1", forkName)
	defer func() {
		testDeleteRepository(t, ctx.Session, "user1", forkName)
	}()

	// create a pull request with some changes
	branchName := "master"
	if deleteBranch {
		branchName = "new_branch_1"
		testEditFileToNewBranch(t, ctx.Session, "user1", forkName, "master", branchName, "README.md", "Hello, World (Edited)\n")
	} else {
		testEditFile(t, ctx.Session, "user1", forkName, "master", "README.md", "Hello, World (Edited)\n")
	}
	testPullCreate(t, ctx.Session, "user1", forkName, false, "master", branchName, "Indexer notifier test pull")

	baseRepo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{OwnerName: "user2", Name: "repo1"})
	forkedRepo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{OwnerName: "user1", Name: forkName})
	pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{
		BaseRepoID: baseRepo.ID,
		BaseBranch: "master",
		HeadRepoID: forkedRepo.ID,
		HeadBranch: branchName,
	})

	if deleteBranch {
		// check if new branch exists
		forkedGitRepo, err := gitrepo.OpenRepository(db.DefaultContext, forkedRepo)
		require.NoError(t, err)
		newBranch, err := forkedGitRepo.GetBranch(branchName)
		require.NoError(t, err)
		assert.NotNil(t, newBranch)
		forkedGitRepo.Close()
	}

	// schedule pull request for automerge
	if withAPI {
		mergePullRequestForm := forms.MergePullRequestForm{
			MergeMessageField:      "auto merge test",
			Do:                     string(repo_model.MergeStyleMerge),
			MergeWhenChecksSucceed: true,
			DeleteBranchAfterMerge: deleteBranch,
		}

		// first time scheduling an automerge pull request, should return a 201
		ctx.ExpectedCode = http.StatusCreated
		doAPIMergePullRequestForm(t, ctx, "user2", "repo1", pr.Index, &mergePullRequestForm)

		// second time scheduling an automerge pull request, should return a 409
		ctx.ExpectedCode = http.StatusConflict
		doAPIMergePullRequestForm(t, ctx, "user2", "repo1", pr.Index, &mergePullRequestForm)
	} else {
		mergePullRequestForm := map[string]string{
			"merge_message_field":       "auto merge test",
			"do":                        string(repo_model.MergeStyleMerge),
			"merge_when_checks_succeed": "true",
			"delete_branch_after_merge": strconv.FormatBool(deleteBranch),
		}

		// first time scheduling an automerge pull request, should return a 200
		testPullMergeForm(t, ctx.Session, http.StatusOK, "user2", "repo1", strconv.FormatInt(pr.Index, 10), mergePullRequestForm)

		// second time scheduling an automerge pull request, should delete the previous scheduled automerge and return a 200 again
		testPullMergeForm(t, ctx.Session, http.StatusOK, "user2", "repo1", strconv.FormatInt(pr.Index, 10), mergePullRequestForm)
	}

	// reload PR again
	pr = unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: pr.ID})
	assert.False(t, pr.HasMerged)
	assert.Empty(t, pr.MergedCommitID)

	// update commit status to success, then it should be merged automatically
	baseGitRepo, err := gitrepo.OpenRepository(db.DefaultContext, baseRepo)
	require.NoError(t, err)
	sha, err := baseGitRepo.GetRefCommitID(pr.GetGitRefName())
	require.NoError(t, err)
	masterCommitID, err := baseGitRepo.GetBranchCommitID("master")
	require.NoError(t, err)

	branches, _, err := baseGitRepo.GetBranchNames(0, 100)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"sub-home-md-img-check", "home-md-img-check", "pr-to-update", "branch2", "DefaultBranch", "develop", "feature/1", "master"}, branches)
	baseGitRepo.Close()
	defer func() {
		testResetRepo(t, baseRepo.RepoPath(), "master", masterCommitID)
	}()

	err = commitstatus_service.CreateCommitStatus(db.DefaultContext, baseRepo, user1, sha, &git_model.CommitStatus{
		State:     api.CommitStatusSuccess,
		TargetURL: "https://gitea.com",
		Context:   "gitea/actions",
	})
	require.NoError(t, err)

	// approve PR if necessary
	if approval {
		// reload PR again
		pr = unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: pr.ID})
		assert.False(t, pr.HasMerged)
		assert.Empty(t, pr.MergedCommitID)

		// approve the PR from non-author
		approveSession := loginUser(t, "user2")
		req := NewRequest(t, "GET", fmt.Sprintf("/user2/repo1/pulls/%d", pr.Index))
		resp := approveSession.MakeRequest(t, req, http.StatusOK)
		htmlDoc := NewHTMLParser(t, resp.Body)
		testSubmitReview(t, approveSession, htmlDoc.GetCSRF(), "user2", "repo1", strconv.Itoa(int(pr.Index)), sha, "approve", http.StatusOK)
	}

	// reload PR again
	pr = unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: pr.ID})
	assert.True(t, pr.HasMerged)
	assert.NotEmpty(t, pr.MergedCommitID)

	unittest.AssertNotExistsBean(t, &pull_model.AutoMerge{PullID: pr.ID})

	if deleteBranch {
		// check if new branch got removed
		forkedGitRepo, err := gitrepo.OpenRepository(db.DefaultContext, forkedRepo)
		require.NoError(t, err)
		_, err = forkedGitRepo.GetBranch(branchName)
		require.Error(t, err)
		assert.True(t, git.IsErrBranchNotExist(err))
		forkedGitRepo.Close()
	}
}

func TestPullAutoMergeAfterCommitStatusSucceed(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, giteaURL *url.URL) {
		for _, testCase := range []struct {
			name         string
			forkName     string
			approval     bool
			deleteBranch bool
		}{
			{
				name:         "TestPullAutoMergeAfterCommitStatusSucceed",
				forkName:     "repo1-1",
				approval:     false,
				deleteBranch: false,
			},
			{
				name:         "TestPullAutoMergeAfterCommitStatusSucceedWithBranchDeletion",
				forkName:     "repo1-2",
				approval:     false,
				deleteBranch: true,
			},
			{
				name:         "TestPullAutoMergeAfterCommitStatusSucceedAndApproval",
				forkName:     "repo1-3",
				approval:     true,
				deleteBranch: false,
			},
			{
				name:         "TestPullAutoMergeAfterCommitStatusSucceedAndApprovalWithBranchDeletion",
				forkName:     "repo1-4",
				approval:     true,
				deleteBranch: true,
			},
		} {
			// perform all tests with API and web routes
			for _, withAPI := range []bool{false, true} {
				t.Run(testCase.name, func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()
					protectedBranch := parameterProtectBranch{
						"enable_push":           "true",
						"enable_status_check":   "true",
						"status_check_contexts": "gitea/actions",
					}
					if testCase.approval {
						protectedBranch["required_approvals"] = "1"
					}
					ctx := NewAPITestContext(t, "user2", "repo1", auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)
					doProtectBranch(ctx, "master", protectedBranch)(t)

					ctx = NewAPITestContext(t, "user1", "repo1", auth_model.AccessTokenScopeWriteRepository)
					testPullAutoMergeAfterCommitStatusSucceed(t, ctx, testCase.forkName, testCase.approval, testCase.deleteBranch, withAPI)
				})
			}
		}
	})
}

func TestPullAutoMergeAfterCommitStatusSucceedAndApprovalForAgitFlow(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		// create a pull request
		baseAPITestContext := NewAPITestContext(t, "user2", "repo1", auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)

		dstPath := t.TempDir()

		u.Path = baseAPITestContext.GitPath()
		u.User = url.UserPassword("user2", userPassword)

		t.Run("Clone", doGitClone(dstPath, u))

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

		stderrBuf := &bytes.Buffer{}

		err = git.NewCommand(git.DefaultContext, "push", "origin", "HEAD:refs/for/master", "-o").
			AddDynamicArguments(`topic=test/head2`).
			AddArguments("-o").
			AddDynamicArguments(`title="create a test pull request with agit"`).
			AddArguments("-o").
			AddDynamicArguments(`description="This PR is a test pull request which created with agit"`).
			Run(&git.RunOpts{Dir: dstPath, Stderr: stderrBuf})
		require.NoError(t, err)

		assert.Contains(t, stderrBuf.String(), setting.AppURL+"user2/repo1/pulls/6")

		baseRepo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{OwnerName: "user2", Name: "repo1"})
		pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{
			Flow:       issues_model.PullRequestFlowAGit,
			BaseRepoID: baseRepo.ID,
			BaseBranch: "master",
			HeadRepoID: baseRepo.ID,
			HeadBranch: "user2/test/head2",
		})

		session := loginUser(t, "user1")
		// add protected branch for commit status
		csrf := GetCSRF(t, session, "/user2/repo1/settings/branches")
		// Change master branch to protected
		req := NewRequestWithValues(t, "POST", "/user2/repo1/settings/branches/edit", map[string]string{
			"_csrf":                 csrf,
			"rule_name":             "master",
			"enable_push":           "true",
			"enable_status_check":   "true",
			"status_check_contexts": "gitea/actions",
			"required_approvals":    "1",
		})
		session.MakeRequest(t, req, http.StatusSeeOther)

		user1 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
		// first time insert automerge record, return true
		scheduled, err := automerge.ScheduleAutoMerge(db.DefaultContext, user1, pr, repo_model.MergeStyleMerge, "auto merge test", false)
		require.NoError(t, err)
		assert.True(t, scheduled)

		// second time insert automerge record, return false because it does exist
		scheduled, err = automerge.ScheduleAutoMerge(db.DefaultContext, user1, pr, repo_model.MergeStyleMerge, "auto merge test", false)
		require.Error(t, err)
		assert.False(t, scheduled)

		// reload pr again
		pr = unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: pr.ID})
		assert.False(t, pr.HasMerged)
		assert.Empty(t, pr.MergedCommitID)

		// update commit status to success, then it should be merged automatically
		baseGitRepo, err := gitrepo.OpenRepository(db.DefaultContext, baseRepo)
		require.NoError(t, err)
		sha, err := baseGitRepo.GetRefCommitID(pr.GetGitRefName())
		require.NoError(t, err)
		masterCommitID, err := baseGitRepo.GetBranchCommitID("master")
		require.NoError(t, err)
		baseGitRepo.Close()
		defer func() {
			testResetRepo(t, baseRepo.RepoPath(), "master", masterCommitID)
		}()

		err = commitstatus_service.CreateCommitStatus(db.DefaultContext, baseRepo, user1, sha, &git_model.CommitStatus{
			State:     api.CommitStatusSuccess,
			TargetURL: "https://gitea.com",
			Context:   "gitea/actions",
		})
		require.NoError(t, err)

		// reload pr again
		pr = unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: pr.ID})
		assert.False(t, pr.HasMerged)
		assert.Empty(t, pr.MergedCommitID)

		// approve the PR from non-author
		approveSession := loginUser(t, "user1")
		req = NewRequest(t, "GET", fmt.Sprintf("/user2/repo1/pulls/%d", pr.Index))
		resp := approveSession.MakeRequest(t, req, http.StatusOK)
		htmlDoc := NewHTMLParser(t, resp.Body)
		testSubmitReview(t, approveSession, htmlDoc.GetCSRF(), "user2", "repo1", strconv.Itoa(int(pr.Index)), sha, "approve", http.StatusOK)

		// realod pr again
		pr = unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: pr.ID})
		assert.True(t, pr.HasMerged)
		assert.NotEmpty(t, pr.MergedCommitID)

		unittest.AssertNotExistsBean(t, &pull_model.AutoMerge{PullID: pr.ID})
	})
}

func TestPullDeleteBranchPerms(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, giteaURL *url.URL) {
		user2Session := loginUser(t, "user2")
		user4Session := loginUser(t, "user4")
		testRepoFork(t, user4Session, "user2", "repo1", "user4", "repo1")
		testEditFileToNewBranch(t, user2Session, "user2", "repo1", "master", "base-pr", "README.md", "Hello, World\n(Edited - base PR)\n")

		req := NewRequestWithValues(t, "POST", "/user4/repo1/compare/master...user2/repo1:base-pr", map[string]string{
			"_csrf": GetCSRF(t, user4Session, "/user4/repo1/compare/master...user2/repo1:base-pr"),
			"title": "Testing PR",
		})
		resp := user4Session.MakeRequest(t, req, http.StatusOK)
		elem := strings.Split(test.RedirectURL(resp), "/")

		req = NewRequestWithValues(t, "POST", "/user4/repo1/pulls/"+elem[4]+"/merge", map[string]string{
			"_csrf":                     GetCSRF(t, user4Session, "/user4/repo1/pulls/"+elem[4]),
			"do":                        "merge",
			"delete_branch_after_merge": "on",
		})
		user4Session.MakeRequest(t, req, http.StatusOK)

		flashCookie := user4Session.GetCookie(forgejo_context.CookieNameFlash)
		assert.NotNil(t, flashCookie)
		assert.Equal(t, "error%3DYou%2Bdon%2527t%2Bhave%2Bpermission%2Bto%2Bdelete%2Bthe%2Bhead%2Bbranch.", flashCookie.Value)

		// Check that the branch still exist.
		req = NewRequest(t, "GET", "/user2/repo1/src/branch/base-pr")
		user4Session.MakeRequest(t, req, http.StatusOK)
	})
}
