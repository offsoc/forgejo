// Copyright 2017 The Gitea Authors. All rights reserved.
// Copyright 2024 The Forgejo Authors c/o Codeberg e.V.. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"regexp"
	"strings"
	"testing"

	"forgejo.org/models/db"
	repo_model "forgejo.org/models/repo"
	unit_model "forgejo.org/models/unit"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/git"
	"forgejo.org/modules/graceful"
	"forgejo.org/modules/test"
	repo_service "forgejo.org/services/repository"
	files_service "forgejo.org/services/repository/files"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testPullCreate(t *testing.T, session *TestSession, user, repo string, toSelf bool, targetBranch, sourceBranch, title string) *httptest.ResponseRecorder {
	req := NewRequest(t, "GET", path.Join(user, repo))
	resp := session.MakeRequest(t, req, http.StatusOK)

	// Click the PR button to create a pull
	htmlDoc := NewHTMLParser(t, resp.Body)
	link, exists := htmlDoc.doc.Find("#new-pull-request").Attr("href")
	assert.True(t, exists, "The template has changed")

	targetUser := strings.Split(link, "/")[1]
	if toSelf && targetUser != user {
		link = strings.Replace(link, targetUser, user, 1)
	}

	// get main out of /user/project/main...some:other/branch
	defaultBranch := regexp.MustCompile(`^.*/(.*)\.\.\.`).FindStringSubmatch(link)[1]
	if targetBranch != defaultBranch {
		link = strings.Replace(link, defaultBranch+"...", targetBranch+"...", 1)
	}
	if sourceBranch != defaultBranch {
		if targetUser == user {
			link = strings.Replace(link, "..."+defaultBranch, "..."+sourceBranch, 1)
		} else {
			link = strings.Replace(link, ":"+defaultBranch, ":"+sourceBranch, 1)
		}
	}

	req = NewRequest(t, "GET", link)
	resp = session.MakeRequest(t, req, http.StatusOK)

	// Submit the form for creating the pull
	htmlDoc = NewHTMLParser(t, resp.Body)
	link, exists = htmlDoc.doc.Find("form.ui.form").Attr("action")
	assert.True(t, exists, "The template has changed")
	req = NewRequestWithValues(t, "POST", link, map[string]string{
		"_csrf": htmlDoc.GetCSRF(),
		"title": title,
	})
	resp = session.MakeRequest(t, req, http.StatusOK)
	return resp
}

func testPullCreateDirectly(t *testing.T, session *TestSession, baseRepoOwner, baseRepoName, baseBranch, headRepoOwner, headRepoName, headBranch, title string) *httptest.ResponseRecorder {
	headCompare := headBranch
	if headRepoOwner != "" {
		if headRepoName != "" {
			headCompare = fmt.Sprintf("%s/%s:%s", headRepoOwner, headRepoName, headBranch)
		} else {
			headCompare = fmt.Sprintf("%s:%s", headRepoOwner, headBranch)
		}
	}
	req := NewRequest(t, "GET", fmt.Sprintf("/%s/%s/compare/%s...%s", baseRepoOwner, baseRepoName, baseBranch, headCompare))
	resp := session.MakeRequest(t, req, http.StatusOK)

	// Submit the form for creating the pull
	htmlDoc := NewHTMLParser(t, resp.Body)
	link, exists := htmlDoc.doc.Find("form.ui.form").Attr("action")
	assert.True(t, exists, "The template has changed")
	req = NewRequestWithValues(t, "POST", link, map[string]string{
		"_csrf": htmlDoc.GetCSRF(),
		"title": title,
	})
	resp = session.MakeRequest(t, req, http.StatusOK)
	return resp
}

func TestPullCreate(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		session := loginUser(t, "user1")
		testRepoFork(t, session, "user2", "repo1", "user1", "repo1")
		testEditFile(t, session, "user1", "repo1", "master", "README.md", "Hello, World (Edited)\n")
		resp := testPullCreate(t, session, "user1", "repo1", false, "master", "master", "This is a pull title")

		// check the redirected URL
		url := test.RedirectURL(resp)
		assert.Regexp(t, "^/user2/repo1/pulls/[0-9]*$", url)

		// check .diff can be accessed and matches performed change
		req := NewRequest(t, "GET", url+".diff")
		resp = session.MakeRequest(t, req, http.StatusOK)
		assert.Regexp(t, `\+Hello, World \(Edited\)`, resp.Body)
		assert.Regexp(t, "^diff", resp.Body)
		assert.NotRegexp(t, "diff.*diff", resp.Body) // not two diffs, just one

		// check .patch can be accessed and matches performed change
		req = NewRequest(t, "GET", url+".patch")
		resp = session.MakeRequest(t, req, http.StatusOK)
		assert.Regexp(t, `\+Hello, World \(Edited\)`, resp.Body)
		assert.Regexp(t, "diff", resp.Body)
		assert.Regexp(t, `Subject: \[PATCH\] Update README.md`, resp.Body)
		assert.NotRegexp(t, "diff.*diff", resp.Body) // not two diffs, just one
	})
}

func TestPullCreateWithPullTemplate(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		baseUser := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
		forkUser := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

		templateCandidates := []string{
			".forgejo/PULL_REQUEST_TEMPLATE.md",
			".forgejo/pull_request_template.md",
			".gitea/PULL_REQUEST_TEMPLATE.md",
			".gitea/pull_request_template.md",
			".github/PULL_REQUEST_TEMPLATE.md",
			".github/pull_request_template.md",
		}

		createBaseRepo := func(t *testing.T, templateFiles []string, message string) (*repo_model.Repository, func()) {
			t.Helper()

			changeOps := make([]*files_service.ChangeRepoFile, len(templateFiles))
			for i, template := range templateFiles {
				changeOps[i] = &files_service.ChangeRepoFile{
					Operation:     "create",
					TreePath:      template,
					ContentReader: strings.NewReader(message + " " + template),
				}
			}

			repo, _, deferrer := tests.CreateDeclarativeRepo(t, baseUser, "", nil, nil, changeOps)

			return repo, deferrer
		}

		testPullPreview := func(t *testing.T, session *TestSession, user, repo, message string) {
			t.Helper()

			req := NewRequest(t, "GET", path.Join(user, repo))
			resp := session.MakeRequest(t, req, http.StatusOK)

			// Click the PR button to create a pull
			htmlDoc := NewHTMLParser(t, resp.Body)
			link, exists := htmlDoc.doc.Find("#new-pull-request").Attr("href")
			assert.True(t, exists, "The template has changed")

			// Load the pull request preview
			req = NewRequest(t, "GET", link)
			resp = session.MakeRequest(t, req, http.StatusOK)

			// Check that the message from the template is present.
			htmlDoc = NewHTMLParser(t, resp.Body)
			pullRequestMessage := htmlDoc.doc.Find("textarea[placeholder*='comment']").Text()
			assert.Equal(t, message, pullRequestMessage)
		}

		for i, template := range templateCandidates {
			t.Run(template, func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				// Create the base repository, with the pull request template added.
				message := fmt.Sprintf("TestPullCreateWithPullTemplate/%s", template)
				baseRepo, deferrer := createBaseRepo(t, []string{template}, message)
				defer deferrer()

				// Fork the repository
				session := loginUser(t, forkUser.Name)
				testRepoFork(t, session, baseUser.Name, baseRepo.Name, forkUser.Name, baseRepo.Name)
				forkedRepo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{OwnerID: forkUser.ID, Name: baseRepo.Name})

				// Apply a change to the fork
				err := createOrReplaceFileInBranch(forkUser, forkedRepo, "README.md", forkedRepo.DefaultBranch, fmt.Sprintf("Hello, World (%d)\n", i))
				require.NoError(t, err)

				testPullPreview(t, session, forkUser.Name, forkedRepo.Name, message+" "+template)
			})
		}

		t.Run("multiple template options", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			// Create the base repository, with the pull request template added.
			message := "TestPullCreateWithPullTemplate/multiple"
			baseRepo, deferrer := createBaseRepo(t, templateCandidates, message)
			defer deferrer()

			// Fork the repository
			session := loginUser(t, forkUser.Name)
			testRepoFork(t, session, baseUser.Name, baseRepo.Name, forkUser.Name, baseRepo.Name)
			forkedRepo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{OwnerID: forkUser.ID, Name: baseRepo.Name})

			// Apply a change to the fork
			err := createOrReplaceFileInBranch(forkUser, forkedRepo, "README.md", forkedRepo.DefaultBranch, "Hello, World (%d)\n")
			require.NoError(t, err)

			// Unlike issues, where all candidates are considered and shown, for
			// pull request, there's a priority: if there are multiple
			// templates, only the highest priority one is used.
			testPullPreview(t, session, forkUser.Name, forkedRepo.Name, message+" .forgejo/PULL_REQUEST_TEMPLATE.md")
		})
	})
}

func TestPullCreate_TitleEscape(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		session := loginUser(t, "user1")
		testRepoFork(t, session, "user2", "repo1", "user1", "repo1")
		testEditFile(t, session, "user1", "repo1", "master", "README.md", "Hello, World (Edited)\n")
		resp := testPullCreate(t, session, "user1", "repo1", false, "master", "master", "<i>XSS PR</i>")

		// check the redirected URL
		url := test.RedirectURL(resp)
		assert.Regexp(t, "^/user2/repo1/pulls/[0-9]*$", url)

		// Edit title
		req := NewRequest(t, "GET", url)
		resp = session.MakeRequest(t, req, http.StatusOK)
		htmlDoc := NewHTMLParser(t, resp.Body)
		editTestTitleURL, exists := htmlDoc.doc.Find(".button-row button[data-update-url]").First().Attr("data-update-url")
		assert.True(t, exists, "The template has changed")

		req = NewRequestWithValues(t, "POST", editTestTitleURL, map[string]string{
			"_csrf": htmlDoc.GetCSRF(),
			"title": "<u>XSS PR</u>",
		})
		session.MakeRequest(t, req, http.StatusOK)

		req = NewRequest(t, "GET", url)
		resp = session.MakeRequest(t, req, http.StatusOK)
		htmlDoc = NewHTMLParser(t, resp.Body)
		titleHTML, err := htmlDoc.doc.Find(".comment-list .timeline-item.event .text b").First().Html()
		require.NoError(t, err)
		assert.Equal(t, "<strike>&lt;i&gt;XSS PR&lt;/i&gt;</strike>", titleHTML)
		titleHTML, err = htmlDoc.doc.Find(".comment-list .timeline-item.event .text b").Next().Html()
		require.NoError(t, err)
		assert.Equal(t, "&lt;u&gt;XSS PR&lt;/u&gt;", titleHTML)
	})
}

func testUIDeleteBranch(t *testing.T, session *TestSession, ownerName, repoName, branchName string) {
	relURL := "/" + path.Join(ownerName, repoName, "branches")
	req := NewRequest(t, "GET", relURL)
	resp := session.MakeRequest(t, req, http.StatusOK)
	htmlDoc := NewHTMLParser(t, resp.Body)

	req = NewRequestWithValues(t, "POST", relURL+"/delete", map[string]string{
		"_csrf": htmlDoc.GetCSRF(),
		"name":  branchName,
	})
	session.MakeRequest(t, req, http.StatusOK)
}

func testDeleteRepository(t *testing.T, session *TestSession, ownerName, repoName string) {
	relURL := "/" + path.Join(ownerName, repoName, "settings")
	req := NewRequest(t, "GET", relURL)
	resp := session.MakeRequest(t, req, http.StatusOK)
	htmlDoc := NewHTMLParser(t, resp.Body)

	req = NewRequestWithValues(t, "POST", relURL+"?action=delete", map[string]string{
		"_csrf":     htmlDoc.GetCSRF(),
		"repo_name": fmt.Sprintf("%s/%s", ownerName, repoName),
	})
	session.MakeRequest(t, req, http.StatusSeeOther)
}

func TestPullBranchDelete(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		defer tests.PrepareTestEnv(t)()

		session := loginUser(t, "user1")
		testRepoFork(t, session, "user2", "repo1", "user1", "repo1")
		testCreateBranch(t, session, "user1", "repo1", "branch/master", "master1", http.StatusSeeOther)
		testEditFile(t, session, "user1", "repo1", "master1", "README.md", "Hello, World (Edited)\n")
		resp := testPullCreate(t, session, "user1", "repo1", false, "master", "master1", "This is a pull title")

		// check the redirected URL
		url := test.RedirectURL(resp)
		assert.Regexp(t, "^/user2/repo1/pulls/[0-9]*$", url)
		req := NewRequest(t, "GET", url)
		session.MakeRequest(t, req, http.StatusOK)

		// delete head branch and confirm pull page is ok
		testUIDeleteBranch(t, session, "user1", "repo1", "master1")
		req = NewRequest(t, "GET", url)
		session.MakeRequest(t, req, http.StatusOK)

		// delete head repository and confirm pull page is ok
		testDeleteRepository(t, session, "user1", "repo1")
		req = NewRequest(t, "GET", url)
		session.MakeRequest(t, req, http.StatusOK)
	})
}

func TestRecentlyPushed(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		session := loginUser(t, "user1")
		testRepoFork(t, session, "user2", "repo1", "user1", "repo1")

		testCreateBranch(t, session, "user1", "repo1", "branch/master", "recent-push", http.StatusSeeOther)
		testEditFile(t, session, "user1", "repo1", "recent-push", "README.md", "Hello recently!\n")

		testCreateBranch(t, session, "user2", "repo1", "branch/master", "recent-push-base", http.StatusSeeOther)
		testEditFile(t, session, "user2", "repo1", "recent-push-base", "README.md", "Hello, recently, from base!\n")

		baseRepo, err := repo_model.GetRepositoryByOwnerAndName(db.DefaultContext, "user2", "repo1")
		require.NoError(t, err)
		repo, err := repo_model.GetRepositoryByOwnerAndName(db.DefaultContext, "user1", "repo1")
		require.NoError(t, err)

		enablePRs := func(t *testing.T, repo *repo_model.Repository) {
			t.Helper()

			err := repo_service.UpdateRepositoryUnits(db.DefaultContext, repo,
				[]repo_model.RepoUnit{{
					RepoID: repo.ID,
					Type:   unit_model.TypePullRequests,
					Config: &repo_model.PullRequestsConfig{
						AllowMerge:  true,
						AllowSquash: true,
					},
				}},
				nil)
			require.NoError(t, err)
		}

		disablePRs := func(t *testing.T, repo *repo_model.Repository) {
			t.Helper()

			err := repo_service.UpdateRepositoryUnits(db.DefaultContext, repo, nil,
				[]unit_model.Type{unit_model.TypePullRequests})
			require.NoError(t, err)
		}

		testBanner := func(t *testing.T) {
			t.Helper()

			req := NewRequest(t, "GET", "/user1/repo1")
			resp := session.MakeRequest(t, req, http.StatusOK)
			htmlDoc := NewHTMLParser(t, resp.Body)

			message := strings.TrimSpace(htmlDoc.Find(".ui.message").Text())
			link, _ := htmlDoc.Find(".ui.message a").Attr("href")
			expectedMessage := "You pushed on branch recent-push"

			assert.Contains(t, message, expectedMessage)
			assert.Equal(t, "/user1/repo1/src/branch/recent-push", link)
		}

		// Test that there's a recently pushed branches banner, and it contains
		// a link to the branch.
		t.Run("recently-pushed-banner", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			testBanner(t)
		})

		// Test that it is still there if the fork has PRs disabled, but the
		// base repo still has them enabled.
		t.Run("with-fork-prs-disabled", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			defer func() {
				enablePRs(t, repo)
			}()

			disablePRs(t, repo)
			testBanner(t)
		})

		// Test that it is still there if the fork has PRs enabled, but the base
		// repo does not.
		t.Run("with-base-prs-disabled", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			defer func() {
				enablePRs(t, baseRepo)
			}()

			disablePRs(t, baseRepo)
			testBanner(t)
		})

		// Test that the banner is not present if both the base and current
		// repo have PRs disabled.
		t.Run("with-prs-disabled", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			defer func() {
				enablePRs(t, baseRepo)
				enablePRs(t, repo)
			}()

			disablePRs(t, repo)
			disablePRs(t, baseRepo)

			req := NewRequest(t, "GET", "/user1/repo1")
			resp := session.MakeRequest(t, req, http.StatusOK)
			htmlDoc := NewHTMLParser(t, resp.Body)
			htmlDoc.AssertElement(t, ".ui.message", false)
		})

		// Test that visiting the base repo has the banner too, and includes
		// recent push notifications from both the fork, and the base repo.
		t.Run("on the base repo", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			// Count recently pushed branches on the fork
			req := NewRequest(t, "GET", "/user1/repo1")
			resp := session.MakeRequest(t, req, http.StatusOK)
			htmlDoc := NewHTMLParser(t, resp.Body)
			htmlDoc.AssertElement(t, ".ui.message", true)

			// Count recently pushed branches on the base repo
			req = NewRequest(t, "GET", "/user2/repo1")
			resp = session.MakeRequest(t, req, http.StatusOK)
			htmlDoc = NewHTMLParser(t, resp.Body)
			messageCountOnBase := htmlDoc.Find(".ui.message").Length()

			// We have two messages on the base: one from the fork, one on the
			// base itself.
			assert.Equal(t, 2, messageCountOnBase)
		})

		// Test that the banner's links point to the right repos
		t.Run("link validity", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			// We're testing against the origin repo, because that has both
			// local branches, and another from a fork, so we can test both in
			// one test!

			req := NewRequest(t, "GET", "/user2/repo1")
			resp := session.MakeRequest(t, req, http.StatusOK)
			htmlDoc := NewHTMLParser(t, resp.Body)
			messages := htmlDoc.Find(".ui.message")

			prButtons := messages.Find("a[role='button']")
			branchLinks := messages.Find("a[href*='/src/branch/']")

			// ** base repo tests **
			basePRLink, _ := prButtons.First().Attr("href")
			baseBranchLink, _ := branchLinks.First().Attr("href")
			baseBranchName := branchLinks.First().Text()

			// branch in the same repo does not have a `user/repo:` qualifier.
			assert.Equal(t, "recent-push-base", baseBranchName)
			// branch link points to the same repo
			assert.Equal(t, "/user2/repo1/src/branch/recent-push-base", baseBranchLink)
			// PR link compares against the correct rep, and unqualified branch name
			assert.Equal(t, "/user2/repo1/compare/master...recent-push-base", basePRLink)

			// ** forked repo tests **
			forkPRLink, _ := prButtons.Last().Attr("href")
			forkBranchLink, _ := branchLinks.Last().Attr("href")
			forkBranchName := branchLinks.Last().Text()

			// branch in the forked repo has a `user/repo:` qualifier.
			assert.Equal(t, "user1/repo1:recent-push", forkBranchName)
			// branch link points to the forked repo
			assert.Equal(t, "/user1/repo1/src/branch/recent-push", forkBranchLink)
			// PR link compares against the correct rep, and qualified branch name
			assert.Equal(t, "/user2/repo1/compare/master...user1/repo1:recent-push", forkPRLink)
		})

		t.Run("unrelated branches are not shown", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			// Create a new branch with no relation to the default branch.
			// 1. Create a new Tree object
			cmd := git.NewCommand(db.DefaultContext, "write-tree")
			treeID, _, gitErr := cmd.RunStdString(&git.RunOpts{Dir: repo.RepoPath()})
			require.NoError(t, gitErr)
			treeID = strings.TrimSpace(treeID)
			// 2. Create a new (empty) commit
			cmd = git.NewCommand(db.DefaultContext, "commit-tree", "-m", "Initial orphan commit").AddDynamicArguments(treeID)
			commitID, _, gitErr := cmd.RunStdString(&git.RunOpts{Dir: repo.RepoPath()})
			require.NoError(t, gitErr)
			commitID = strings.TrimSpace(commitID)
			// 3. Create a new ref pointing to the orphaned commit
			cmd = git.NewCommand(db.DefaultContext, "update-ref", "refs/heads/orphan1").AddDynamicArguments(commitID)
			_, _, gitErr = cmd.RunStdString(&git.RunOpts{Dir: repo.RepoPath()})
			require.NoError(t, gitErr)
			// 4. Sync the git repo to the database
			syncErr := repo_service.AddAllRepoBranchesToSyncQueue(graceful.GetManager().ShutdownContext())
			require.NoError(t, syncErr)
			// 5. Add a fresh commit, so that FindRecentlyPushedBranches has
			// something to find.
			owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: "user1"})
			changeResp, err := files_service.ChangeRepoFiles(git.DefaultContext, repo, owner,
				&files_service.ChangeRepoFilesOptions{
					Files: []*files_service.ChangeRepoFile{
						{
							Operation:     "create",
							TreePath:      "README.md",
							ContentReader: strings.NewReader("a readme file"),
						},
					},
					Message:   "Add README.md",
					OldBranch: "orphan1",
					NewBranch: "orphan1",
				})
			require.NoError(t, err)
			assert.NotEmpty(t, changeResp)

			// Check that we only have 1 message on the main repo, the orphaned
			// one is not shown.
			req := NewRequest(t, "GET", "/user1/repo1")
			resp := session.MakeRequest(t, req, http.StatusOK)
			htmlDoc := NewHTMLParser(t, resp.Body)

			htmlDoc.AssertElement(t, ".ui.message", true)
			link, _ := htmlDoc.Find(".ui.message a[href*='/src/branch/']").Attr("href")
			assert.Equal(t, "/user1/repo1/src/branch/recent-push", link)
		})

		// Test that visiting the base repo does not show any banner if
		// the branches have corresponding PRs (open or merged)
		t.Run("branches with merged or open PRs are not shown", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			respChildPR := testPullCreateDirectly(t, session, "user2", "repo1", "master", "user1", "repo1", "recent-push", "Child Pull Request")
			elemChildPR := strings.Split(test.RedirectURL(respChildPR), "/")
			assert.Equal(t, "user2", elemChildPR[1])
			assert.Equal(t, "repo1", elemChildPR[2])
			assert.Equal(t, "pulls", elemChildPR[3])
			session2 := loginUser(t, "user2")
			// Merge the PR from the fork
			testPullMerge(t, session2, elemChildPR[1], elemChildPR[2], elemChildPR[4], repo_model.MergeStyleSquash, false)

			respBasePR := testPullCreate(t, session, "user2", "repo1", true, "master", "recent-push-base", "Base Pull Request")
			elemBasePR := strings.Split(test.RedirectURL(respBasePR), "/")
			assert.Equal(t, "pulls", elemBasePR[3])
			// Leave the PR from the base repo open (it conflicts with the PR from the fork anyway)

			// Count recently pushed branches on the base repo
			req := NewRequest(t, "GET", "/user2/repo1")
			resp := session.MakeRequest(t, req, http.StatusOK)
			htmlDoc := NewHTMLParser(t, resp.Body)

			messages := htmlDoc.Find(".ui.message")

			// None of the branches should be shown, as they have either already been merged already,
			// or have an open PR, so it doesn't make sense to make a new PR for any of them.
			assert.Equal(t, 0, messages.Length())
		})
	})
}

/*
Setup:
The base repository is: user2/repo1
Fork repository to: user1/repo1
Push extra commit to: user2/repo1, which changes README.md
Create a PR on user1/repo1

Test checks:
Check if pull request can be created from base to the fork repository.
*/
func TestPullCreatePrFromBaseToFork(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		sessionFork := loginUser(t, "user1")
		testRepoFork(t, sessionFork, "user2", "repo1", "user1", "repo1")

		// Edit base repository
		sessionBase := loginUser(t, "user2")
		testEditFile(t, sessionBase, "user2", "repo1", "master", "README.md", "Hello, World (Edited)\n")

		// Create a PR
		resp := testPullCreateDirectly(t, sessionFork, "user1", "repo1", "master", "user2", "repo1", "master", "This is a pull title")
		// check the redirected URL
		url := test.RedirectURL(resp)
		assert.Regexp(t, "^/user1/repo1/pulls/[0-9]*$", url)
	})
}
