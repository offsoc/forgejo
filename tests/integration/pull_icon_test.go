// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"forgejo.org/models/db"
	issues_model "forgejo.org/models/issues"
	repo_model "forgejo.org/models/repo"
	unit_model "forgejo.org/models/unit"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/git"
	issue_service "forgejo.org/services/issue"
	pull_service "forgejo.org/services/pull"
	files_service "forgejo.org/services/repository/files"
	"forgejo.org/tests"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPullRequestIcons(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
		repo, _, f := tests.CreateDeclarativeRepo(t, user, "pr-icons", []unit_model.Type{unit_model.TypeCode, unit_model.TypePullRequests}, nil, nil)
		defer f()

		session := loginUser(t, user.LoginName)

		// Individual PRs
		t.Run("Open", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			pull := createOpenPullRequest(db.DefaultContext, t, user, repo)
			testPullRequestIcon(t, session, pull, "green", "octicon-git-pull-request")
		})

		t.Run("WIP (Open)", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			pull := createOpenWipPullRequest(db.DefaultContext, t, user, repo)
			testPullRequestIcon(t, session, pull, "grey", "octicon-git-pull-request-draft")
		})

		t.Run("Closed", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			pull := createClosedPullRequest(db.DefaultContext, t, user, repo)
			testPullRequestIcon(t, session, pull, "red", "octicon-git-pull-request-closed")
		})

		t.Run("WIP (Closed)", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			pull := createClosedWipPullRequest(db.DefaultContext, t, user, repo)
			testPullRequestIcon(t, session, pull, "red", "octicon-git-pull-request-closed")
		})

		t.Run("Merged", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			pull := createMergedPullRequest(db.DefaultContext, t, user, repo)
			testPullRequestIcon(t, session, pull, "purple", "octicon-git-merge")
		})

		// List
		req := NewRequest(t, "GET", repo.HTMLURL()+"/pulls?state=all")
		resp := session.MakeRequest(t, req, http.StatusOK)
		doc := NewHTMLParser(t, resp.Body)

		t.Run("List Open", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			testPullRequestListIcon(t, doc, "open", "green", "octicon-git-pull-request")
		})

		t.Run("List WIP (Open)", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			testPullRequestListIcon(t, doc, "open-wip", "grey", "octicon-git-pull-request-draft")
		})

		t.Run("List Closed", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			testPullRequestListIcon(t, doc, "closed", "red", "octicon-git-pull-request-closed")
		})

		t.Run("List Closed (WIP)", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			testPullRequestListIcon(t, doc, "closed-wip", "red", "octicon-git-pull-request-closed")
		})

		t.Run("List Merged", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			testPullRequestListIcon(t, doc, "merged", "purple", "octicon-git-merge")
		})
	})
}

func testPullRequestIcon(t *testing.T, session *TestSession, pr *issues_model.PullRequest, expectedColor, expectedIcon string) {
	req := NewRequest(t, "GET", pr.Issue.HTMLURL())
	resp := session.MakeRequest(t, req, http.StatusOK)
	doc := NewHTMLParser(t, resp.Body)
	doc.AssertElement(t, fmt.Sprintf("div.issue-state-label.%s > svg.%s", expectedColor, expectedIcon), true)

	req = NewRequest(t, "GET", pr.BaseRepo.HTMLURL()+"/branches")
	resp = session.MakeRequest(t, req, http.StatusOK)
	doc = NewHTMLParser(t, resp.Body)
	doc.AssertElement(t, fmt.Sprintf(`a[href="/%s/pulls/%d"].%s > svg.%s`, pr.BaseRepo.FullName(), pr.Issue.Index, expectedColor, expectedIcon), true)
}

func testPullRequestListIcon(t *testing.T, doc *HTMLDoc, name, expectedColor, expectedIcon string) {
	sel := doc.doc.Find("div#issue-list > div.flex-item").
		FilterFunction(func(_ int, selection *goquery.Selection) bool {
			return selection.Find(fmt.Sprintf(`div.flex-item-icon > svg.%s.%s`, expectedColor, expectedIcon)).Length() == 1 &&
				strings.HasSuffix(selection.Find("a.issue-title").Text(), name)
		})

	assert.Equal(t, 1, sel.Length())
}

func createOpenPullRequest(ctx context.Context, t *testing.T, user *user_model.User, repo *repo_model.Repository) *issues_model.PullRequest {
	pull := createPullRequest(t, user, repo, "branch-open", "open")

	assert.False(t, pull.Issue.IsClosed)
	assert.False(t, pull.HasMerged)
	assert.False(t, pull.IsWorkInProgress(ctx))

	return pull
}

func createOpenWipPullRequest(ctx context.Context, t *testing.T, user *user_model.User, repo *repo_model.Repository) *issues_model.PullRequest {
	pull := createPullRequest(t, user, repo, "branch-open-wip", "open-wip")

	err := issue_service.ChangeTitle(ctx, pull.Issue, user, "WIP: "+pull.Issue.Title)
	require.NoError(t, err)

	assert.False(t, pull.Issue.IsClosed)
	assert.False(t, pull.HasMerged)
	assert.True(t, pull.IsWorkInProgress(ctx))

	return pull
}

func createClosedPullRequest(ctx context.Context, t *testing.T, user *user_model.User, repo *repo_model.Repository) *issues_model.PullRequest {
	pull := createPullRequest(t, user, repo, "branch-closed", "closed")

	err := issue_service.ChangeStatus(ctx, pull.Issue, user, "", true)
	require.NoError(t, err)

	assert.True(t, pull.Issue.IsClosed)
	assert.False(t, pull.HasMerged)
	assert.False(t, pull.IsWorkInProgress(ctx))

	return pull
}

func createClosedWipPullRequest(ctx context.Context, t *testing.T, user *user_model.User, repo *repo_model.Repository) *issues_model.PullRequest {
	pull := createPullRequest(t, user, repo, "branch-closed-wip", "closed-wip")

	err := issue_service.ChangeTitle(ctx, pull.Issue, user, "WIP: "+pull.Issue.Title)
	require.NoError(t, err)

	err = issue_service.ChangeStatus(ctx, pull.Issue, user, "", true)
	require.NoError(t, err)

	assert.True(t, pull.Issue.IsClosed)
	assert.False(t, pull.HasMerged)
	assert.True(t, pull.IsWorkInProgress(ctx))

	return pull
}

func createMergedPullRequest(ctx context.Context, t *testing.T, user *user_model.User, repo *repo_model.Repository) *issues_model.PullRequest {
	pull := createPullRequest(t, user, repo, "branch-merged", "merged")

	gitRepo, err := git.OpenRepository(ctx, repo.RepoPath())
	defer gitRepo.Close()

	require.NoError(t, err)

	err = pull_service.Merge(ctx, pull, user, gitRepo, repo_model.MergeStyleMerge, pull.HeadCommitID, "merge", false)
	require.NoError(t, err)

	assert.False(t, pull.Issue.IsClosed)
	assert.True(t, pull.CanAutoMerge())
	assert.False(t, pull.IsWorkInProgress(ctx))

	return pull
}

func createPullRequest(t *testing.T, user *user_model.User, repo *repo_model.Repository, branch, title string) *issues_model.PullRequest {
	_, err := files_service.ChangeRepoFiles(git.DefaultContext, repo, user, &files_service.ChangeRepoFilesOptions{
		Files: []*files_service.ChangeRepoFile{
			{
				Operation:     "update",
				TreePath:      "README.md",
				ContentReader: strings.NewReader("Update README"),
			},
		},
		Message:   "Update README",
		OldBranch: "main",
		NewBranch: branch,
		Author: &files_service.IdentityOptions{
			Name:  user.Name,
			Email: user.Email,
		},
		Committer: &files_service.IdentityOptions{
			Name:  user.Name,
			Email: user.Email,
		},
		Dates: &files_service.CommitDateOptions{
			Author:    time.Now(),
			Committer: time.Now(),
		},
	})

	require.NoError(t, err)

	pullIssue := &issues_model.Issue{
		RepoID:   repo.ID,
		Title:    title,
		PosterID: user.ID,
		Poster:   user,
		IsPull:   true,
	}

	pullRequest := &issues_model.PullRequest{
		HeadRepoID: repo.ID,
		BaseRepoID: repo.ID,
		HeadBranch: branch,
		BaseBranch: "main",
		HeadRepo:   repo,
		BaseRepo:   repo,
		Type:       issues_model.PullRequestGitea,
	}
	err = pull_service.NewPullRequest(git.DefaultContext, repo, pullIssue, nil, nil, pullRequest, nil)
	require.NoError(t, err)

	return pullRequest
}
