// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"forgejo.org/models/db"
	issues_model "forgejo.org/models/issues"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/git"
	issue_service "forgejo.org/services/issue"
	pull_service "forgejo.org/services/pull"
	files_service "forgejo.org/services/repository/files"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIssueTitles(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
		repo, _, f := tests.CreateDeclarativeRepo(t, user, "issue-titles", nil, nil, nil)
		defer f()

		session := loginUser(t, user.LoginName)

		title := "Title :+1: `code`"
		issue1 := createIssue(t, user, repo, title, "Test issue")
		issue2 := createIssue(t, user, repo, title, "Ref #1")

		titleHTML := []string{
			"Title",
			`<span class="emoji" aria-label="thumbs up" data-alias="+1">👍</span>`,
			`<code class="inline-code-block">code</code>`,
		}

		t.Run("Main issue title", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			html := extractHTML(t, session, issue1, "div.issue-title-header > * > h1")
			assertContainsAll(t, titleHTML, html)
		})

		t.Run("Referenced issue comment", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			html := extractHTML(t, session, issue1, "div.timeline > div.timeline-item:nth-child(3) > div.detail > * > a")
			assertContainsAll(t, titleHTML, html)
		})

		t.Run("Dependent issue comment", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			err := issues_model.CreateIssueDependency(db.DefaultContext, user, issue1, issue2)
			require.NoError(t, err)

			html := extractHTML(t, session, issue1, "div.timeline > div:nth-child(3) > div.detail > * > a")
			assertContainsAll(t, titleHTML, html)
		})

		t.Run("Dependent issue sidebar", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			html := extractHTML(t, session, issue1, "div.item.dependency > * > a.title")
			assertContainsAll(t, titleHTML, html)
		})

		t.Run("Referenced pull comment", func(t *testing.T) {
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
				NewBranch: "branch",
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
				Content:  "Closes #1",
				PosterID: user.ID,
				Poster:   user,
				IsPull:   true,
			}

			pullRequest := &issues_model.PullRequest{
				HeadRepoID: repo.ID,
				BaseRepoID: repo.ID,
				HeadBranch: "branch",
				BaseBranch: "main",
				HeadRepo:   repo,
				BaseRepo:   repo,
				Type:       issues_model.PullRequestGitea,
			}

			err = pull_service.NewPullRequest(git.DefaultContext, repo, pullIssue, nil, nil, pullRequest, nil)
			require.NoError(t, err)

			html := extractHTML(t, session, issue1, "div.timeline > div:nth-child(4) > div.detail > * > a")
			assertContainsAll(t, titleHTML, html)
		})
	})
}

func createIssue(t *testing.T, user *user_model.User, repo *repo_model.Repository, title, content string) *issues_model.Issue {
	issue := &issues_model.Issue{
		RepoID:   repo.ID,
		Title:    title,
		Content:  content,
		PosterID: user.ID,
		Poster:   user,
	}

	err := issue_service.NewIssue(db.DefaultContext, repo, issue, nil, nil, nil)
	require.NoError(t, err)

	return issue
}

func extractHTML(t *testing.T, session *TestSession, issue *issues_model.Issue, query string) string {
	req := NewRequest(t, "GET", issue.HTMLURL())
	resp := session.MakeRequest(t, req, http.StatusOK)
	doc := NewHTMLParser(t, resp.Body)
	res, err := doc.doc.Find(query).Html()
	require.NoError(t, err)

	return res
}

func assertContainsAll(t *testing.T, expected []string, actual string) {
	for i := range expected {
		assert.Contains(t, actual, expected[i])
	}
}
