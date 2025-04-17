// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"forgejo.org/models/db"
	git_model "forgejo.org/models/git"
	issues_model "forgejo.org/models/issues"
	unit_model "forgejo.org/models/unit"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/git"
	"forgejo.org/modules/translation"
	gitea_context "forgejo.org/services/context"
	issue_service "forgejo.org/services/issue"
	pull_service "forgejo.org/services/pull"
	repo_service "forgejo.org/services/repository"
	files_service "forgejo.org/services/repository/files"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPullrequestReopen(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
		org26 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 26})

		// Create an base repository.
		baseRepo, _, f := tests.CreateDeclarativeRepo(t, user2, "reopen-base",
			[]unit_model.Type{unit_model.TypePullRequests}, nil, nil,
		)
		defer f()

		// Create a new branch on the base branch, so it can be deleted later.
		_, err := files_service.ChangeRepoFiles(git.DefaultContext, baseRepo, user2, &files_service.ChangeRepoFilesOptions{
			Files: []*files_service.ChangeRepoFile{
				{
					Operation:     "update",
					TreePath:      "README.md",
					ContentReader: strings.NewReader("New README.md"),
				},
			},
			Message:   "Modify README for base",
			OldBranch: "main",
			NewBranch: "base-branch",
			Author: &files_service.IdentityOptions{
				Name:  user2.Name,
				Email: user2.Email,
			},
			Committer: &files_service.IdentityOptions{
				Name:  user2.Name,
				Email: user2.Email,
			},
			Dates: &files_service.CommitDateOptions{
				Author:    time.Now(),
				Committer: time.Now(),
			},
		})
		require.NoError(t, err)

		// Create an head repository.
		headRepo, err := repo_service.ForkRepositoryAndUpdates(git.DefaultContext, user2, org26, repo_service.ForkRepoOptions{
			BaseRepo: baseRepo,
			Name:     "reopen-head",
		})
		require.NoError(t, err)
		assert.NotEmpty(t, headRepo)

		// Add a change to the head repository, so a pull request can be opened.
		_, err = files_service.ChangeRepoFiles(git.DefaultContext, headRepo, user2, &files_service.ChangeRepoFilesOptions{
			Files: []*files_service.ChangeRepoFile{
				{
					Operation:     "update",
					TreePath:      "README.md",
					ContentReader: strings.NewReader("Updated README.md"),
				},
			},
			Message:   "Modify README for head",
			OldBranch: "main",
			NewBranch: "head-branch",
			Author: &files_service.IdentityOptions{
				Name:  user2.Name,
				Email: user2.Email,
			},
			Committer: &files_service.IdentityOptions{
				Name:  user2.Name,
				Email: user2.Email,
			},
			Dates: &files_service.CommitDateOptions{
				Author:    time.Now(),
				Committer: time.Now(),
			},
		})
		require.NoError(t, err)

		// Create the pull request.
		pullIssue := &issues_model.Issue{
			RepoID:   baseRepo.ID,
			Title:    "Testing reopen functionality",
			PosterID: user2.ID,
			Poster:   user2,
			IsPull:   true,
		}
		pullRequest := &issues_model.PullRequest{
			HeadRepoID: headRepo.ID,
			BaseRepoID: baseRepo.ID,
			HeadBranch: "head-branch",
			BaseBranch: "base-branch",
			HeadRepo:   headRepo,
			BaseRepo:   baseRepo,
			Type:       issues_model.PullRequestGitea,
		}
		err = pull_service.NewPullRequest(git.DefaultContext, baseRepo, pullIssue, nil, nil, pullRequest, nil)
		require.NoError(t, err)

		issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{Title: "Testing reopen functionality"})

		// Close the PR.
		err = issue_service.ChangeStatus(db.DefaultContext, issue, user2, "", true)
		require.NoError(t, err)

		session := loginUser(t, "user2")

		reopenPR := func(t *testing.T, expectedStatus int) *httptest.ResponseRecorder {
			t.Helper()

			link := fmt.Sprintf("%s/pulls/%d/comments", baseRepo.FullName(), issue.Index)
			req := NewRequestWithValues(t, "POST", link, map[string]string{
				"_csrf":  GetCSRF(t, session, fmt.Sprintf("%s/pulls/%d", baseRepo.FullName(), issue.Index)),
				"status": "reopen",
			})
			return session.MakeRequest(t, req, expectedStatus)
		}

		restoreBranch := func(t *testing.T, repoName, branchName string, branchID int64) {
			t.Helper()

			link := fmt.Sprintf("/%s/branches", repoName)
			req := NewRequestWithValues(t, "POST", fmt.Sprintf("%s/restore?branch_id=%d&name=%s", link, branchID, branchName), map[string]string{
				"_csrf": GetCSRF(t, session, link),
			})
			session.MakeRequest(t, req, http.StatusOK)

			flashCookie := session.GetCookie(gitea_context.CookieNameFlash)
			assert.NotNil(t, flashCookie)
			assert.Contains(t, flashCookie.Value, "success%3DBranch%2B%2522"+branchName+"%2522%2Bhas%2Bbeen%2Brestored.")
		}

		deleteBranch := func(t *testing.T, repoName, branchName string) {
			t.Helper()

			link := fmt.Sprintf("/%s/branches", repoName)
			req := NewRequestWithValues(t, "POST", fmt.Sprintf("%s/delete?name=%s", link, branchName), map[string]string{
				"_csrf": GetCSRF(t, session, link),
			})
			session.MakeRequest(t, req, http.StatusOK)

			flashCookie := session.GetCookie(gitea_context.CookieNameFlash)
			assert.NotNil(t, flashCookie)
			assert.Contains(t, flashCookie.Value, "success%3DBranch%2B%2522"+branchName+"%2522%2Bhas%2Bbeen%2Bdeleted.")
		}

		type errorJSON struct {
			Error string `json:"errorMessage"`
		}

		t.Run("Base branch deleted", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			branch := unittest.AssertExistsAndLoadBean(t, &git_model.Branch{Name: "base-branch", RepoID: baseRepo.ID})
			defer func() {
				restoreBranch(t, baseRepo.FullName(), branch.Name, branch.ID)
			}()

			deleteBranch(t, baseRepo.FullName(), branch.Name)
			resp := reopenPR(t, http.StatusBadRequest)

			var errorResp errorJSON
			DecodeJSON(t, resp, &errorResp)
			assert.EqualValues(t, translation.NewLocale("en-US").Tr("repo.pulls.reopen_failed.base_branch"), errorResp.Error)
		})

		t.Run("Head branch deleted", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			branch := unittest.AssertExistsAndLoadBean(t, &git_model.Branch{Name: "head-branch", RepoID: headRepo.ID})
			defer func() {
				restoreBranch(t, headRepo.FullName(), branch.Name, branch.ID)
			}()

			deleteBranch(t, headRepo.FullName(), branch.Name)
			resp := reopenPR(t, http.StatusBadRequest)

			var errorResp errorJSON
			DecodeJSON(t, resp, &errorResp)
			assert.EqualValues(t, translation.NewLocale("en-US").Tr("repo.pulls.reopen_failed.head_branch"), errorResp.Error)
		})

		t.Run("Normal", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			reopenPR(t, http.StatusOK)
		})
	})
}
