// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
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
	files_service "forgejo.org/services/repository/files"
	"forgejo.org/tests"

	"github.com/stretchr/testify/require"
)

func TestCodeOwner(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

		// Create the repo.
		repo, _, f := tests.CreateDeclarativeRepo(t, user2, "",
			[]unit_model.Type{unit_model.TypePullRequests}, nil,
			[]*files_service.ChangeRepoFile{
				{
					Operation:     "create",
					TreePath:      "CODEOWNERS",
					ContentReader: strings.NewReader("README.md @user5\ntest-file @user4"),
				},
			},
		)
		defer f()

		dstPath := t.TempDir()
		r := fmt.Sprintf("%suser2/%s.git", u.String(), repo.Name)
		cloneURL, _ := url.Parse(r)
		cloneURL.User = url.UserPassword("user2", userPassword)
		require.NoError(t, git.CloneWithArgs(t.Context(), nil, cloneURL.String(), dstPath, git.CloneRepoOptions{}))

		t.Run("Normal", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			err := os.WriteFile(path.Join(dstPath, "README.md"), []byte("## test content"), 0o666)
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
				Message: "Add README.",
			})
			require.NoError(t, err)

			err = git.NewCommand(git.DefaultContext, "push", "origin", "HEAD:refs/for/main", "-o", "topic=codeowner-normal").Run(&git.RunOpts{Dir: dstPath})
			require.NoError(t, err)

			pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{BaseRepoID: repo.ID, HeadBranch: "user2/codeowner-normal"})
			unittest.AssertExistsIf(t, true, &issues_model.Review{IssueID: pr.IssueID, Type: issues_model.ReviewTypeRequest, ReviewerID: 5})
		})

		t.Run("Forked repository", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			session := loginUser(t, "user1")
			testRepoFork(t, session, user2.Name, repo.Name, "user1", "repo1")

			repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{OwnerName: "user1", Name: "repo1"})

			r := fmt.Sprintf("%suser1/repo1.git", u.String())
			remoteURL, _ := url.Parse(r)
			remoteURL.User = url.UserPassword("user2", userPassword)
			doGitAddRemote(dstPath, "forked", remoteURL)(t)

			err := git.NewCommand(git.DefaultContext, "push", "forked", "HEAD:refs/for/main", "-o", "topic=codeowner-forked").Run(&git.RunOpts{Dir: dstPath})
			require.NoError(t, err)

			pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{BaseRepoID: repo.ID, HeadBranch: "user2/codeowner-forked"})
			unittest.AssertExistsIf(t, false, &issues_model.Review{IssueID: pr.IssueID, Type: issues_model.ReviewTypeRequest, ReviewerID: 5})
		})

		t.Run("Out of date", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			// Push the changes made from the previous subtest.
			require.NoError(t, git.NewCommand(git.DefaultContext, "push", "origin").Run(&git.RunOpts{Dir: dstPath}))

			// Reset the tree to the previous commit.
			require.NoError(t, git.NewCommand(git.DefaultContext, "reset", "--hard", "HEAD~1").Run(&git.RunOpts{Dir: dstPath}))

			err := os.WriteFile(path.Join(dstPath, "test-file"), []byte("## test content"), 0o666)
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
				Message: "Add test-file.",
			})
			require.NoError(t, err)

			err = git.NewCommand(git.DefaultContext, "push", "origin", "HEAD:refs/for/main", "-o", "topic=codeowner-out-of-date").Run(&git.RunOpts{Dir: dstPath})
			require.NoError(t, err)

			pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{BaseRepoID: repo.ID, HeadBranch: "user2/codeowner-out-of-date"})
			unittest.AssertExistsIf(t, true, &issues_model.Review{IssueID: pr.IssueID, Type: issues_model.ReviewTypeRequest, ReviewerID: 4})
			unittest.AssertExistsIf(t, false, &issues_model.Review{IssueID: pr.IssueID, Type: issues_model.ReviewTypeRequest, ReviewerID: 5})
		})
		t.Run("From a forked repository", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			session := loginUser(t, "user1")

			r := fmt.Sprintf("%suser1/repo1.git", u.String())
			remoteURL, _ := url.Parse(r)
			remoteURL.User = url.UserPassword("user1", userPassword)
			doGitAddRemote(dstPath, "forked-2", remoteURL)(t)

			err := git.NewCommand(git.DefaultContext, "push", "forked-2", "HEAD:branch").Run(&git.RunOpts{Dir: dstPath})
			require.NoError(t, err)

			req := NewRequestWithValues(t, "POST", repo.FullName()+"/compare/main...user1/repo1:branch", map[string]string{
				"_csrf": GetCSRF(t, session, repo.FullName()+"/compare/main...user1/repo1:branch"),
				"title": "pull request",
			})
			session.MakeRequest(t, req, http.StatusOK)

			pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{BaseRepoID: repo.ID, HeadBranch: "branch"})
			unittest.AssertExistsIf(t, true, &issues_model.Review{IssueID: pr.IssueID, Type: issues_model.ReviewTypeRequest, ReviewerID: 4})
		})

		t.Run("Codeowner user with no permission", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			// Make repository private, only user2 (owner of repository) has now access to this repository.
			repo.IsPrivate = true
			_, err := db.GetEngine(db.DefaultContext).Cols("is_private").Update(repo)
			require.NoError(t, err)

			err = os.WriteFile(path.Join(dstPath, "README.md"), []byte("## very sensitive info"), 0o666)
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
				Message: "Add secrets to the README.",
			})
			require.NoError(t, err)

			err = git.NewCommand(git.DefaultContext, "push", "origin", "HEAD:refs/for/main", "-o", "topic=codeowner-private").Run(&git.RunOpts{Dir: dstPath})
			require.NoError(t, err)

			// In CODEOWNERS file the codeowner for README.md is user5, but does not have access to this private repository.
			pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{BaseRepoID: repo.ID, HeadBranch: "user2/codeowner-private"})
			unittest.AssertExistsIf(t, false, &issues_model.Review{IssueID: pr.IssueID, Type: issues_model.ReviewTypeRequest, ReviewerID: 5})
		})
	})
}
