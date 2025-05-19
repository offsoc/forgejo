// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	issues_model "forgejo.org/models/issues"
	repo_model "forgejo.org/models/repo"
	unit_model "forgejo.org/models/unit"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/git"
	"forgejo.org/modules/optional"
	files_service "forgejo.org/services/repository/files"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPatchStatus(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
		session := loginUser(t, user2.Name)

		repo, _, f := tests.CreateDeclarativeRepoWithOptions(t, user2, tests.DeclarativeRepoOptions{
			AutoInit:     optional.Some(true),
			EnabledUnits: optional.Some([]unit_model.Type{unit_model.TypeCode}),
			ObjectFormat: optional.Some("sha256"),
			Files: optional.Some([]*files_service.ChangeRepoFile{
				{
					Operation:     "create",
					TreePath:      ".spokeperson",
					ContentReader: strings.NewReader("n0toose"),
				},
			}),
		})
		defer f()

		testRepoFork(t, session, "user2", repo.Name, "org3", "forked-repo")
		forkRepo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{OwnerName: "org3", Name: "forked-repo"})

		u.User = url.UserPassword(user2.Name, userPassword)
		u.Path = repo.FullName()

		// Clone repository.
		dstPath := t.TempDir()
		require.NoError(t, git.Clone(t.Context(), u.String(), dstPath, git.CloneRepoOptions{}))

		// Add `fork` remote.
		u.Path = forkRepo.FullName()
		_, _, err := git.NewCommand(git.DefaultContext, "remote", "add", "fork").AddDynamicArguments(u.String()).RunStdString(&git.RunOpts{Dir: dstPath})
		require.NoError(t, err)

		var normalAGitPR *issues_model.PullRequest

		// Normal pull request, should be mergeable.
		t.Run("Normal", func(t *testing.T) {
			require.NoError(t, git.NewCommand(t.Context(), "switch", "-c", "normal").AddDynamicArguments(repo.DefaultBranch).Run(&git.RunOpts{Dir: dstPath}))
			require.NoError(t, os.WriteFile(filepath.Join(dstPath, "CONTACT"), []byte("n0toose@example.com"), 0o600))
			require.NoError(t, git.NewCommand(t.Context(), "add", "CONTACT").Run(&git.RunOpts{Dir: dstPath}))
			require.NoError(t, git.NewCommand(t.Context(), "commit", "--message=fancy").Run(&git.RunOpts{Dir: dstPath}))

			test := func(t *testing.T, pr *issues_model.PullRequest) {
				t.Helper()

				assert.Empty(t, pr.ConflictedFiles)
				assert.Equal(t, issues_model.PullRequestStatusMergeable, pr.Status)
				assert.Equal(t, 1, pr.CommitsAhead)
				assert.Equal(t, 0, pr.CommitsBehind)
				assert.True(t, pr.Mergeable(t.Context()))
			}

			t.Run("Across repository", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				require.NoError(t, git.NewCommand(t.Context(), "push", "fork", "HEAD:normal").Run(&git.RunOpts{Dir: dstPath}))
				testPullCreateDirectly(t, session, repo.OwnerName, repo.Name, repo.DefaultBranch, forkRepo.OwnerName, forkRepo.Name, "normal", "across repo normal")

				test(t, unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{BaseRepoID: repo.ID, HeadRepoID: forkRepo.ID, HeadBranch: "normal"}, "flow = 0"))
			})

			t.Run("Same repository", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				require.NoError(t, git.NewCommand(t.Context(), "push", "origin", "HEAD:normal").Run(&git.RunOpts{Dir: dstPath}))
				testPullCreateDirectly(t, session, repo.OwnerName, repo.Name, repo.DefaultBranch, repo.OwnerName, repo.Name, "normal", "same repo normal")

				test(t, unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{BaseRepoID: repo.ID, HeadRepoID: repo.ID, HeadBranch: "normal"}, "flow = 0"))
			})

			t.Run("AGit", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				require.NoError(t, git.NewCommand(t.Context(), "push", "origin", "HEAD:refs/for/main", "-o", "topic=normal").Run(&git.RunOpts{Dir: dstPath}))

				normalAGitPR = unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{BaseRepoID: repo.ID, HeadRepoID: repo.ID, HeadBranch: "user2/normal", Flow: issues_model.PullRequestFlowAGit})
				test(t, normalAGitPR)
			})
		})

		// If there's a merge conflict, either on update of the base branch or on
		// creation of the pull request then it should be marked as such.
		t.Run("Conflict", func(t *testing.T) {
			require.NoError(t, git.NewCommand(t.Context(), "switch").AddDynamicArguments(repo.DefaultBranch).Run(&git.RunOpts{Dir: dstPath}))
			require.NoError(t, os.WriteFile(filepath.Join(dstPath, "CONTACT"), []byte("gusted@example.com"), 0o600))
			require.NoError(t, git.NewCommand(t.Context(), "add", "CONTACT").Run(&git.RunOpts{Dir: dstPath}))
			require.NoError(t, git.NewCommand(t.Context(), "commit", "--message=fancy").Run(&git.RunOpts{Dir: dstPath}))
			require.NoError(t, git.NewCommand(t.Context(), "push", "origin", "HEAD:main").Run(&git.RunOpts{Dir: dstPath}))
			require.NoError(t, git.NewCommand(t.Context(), "switch", "normal").Run(&git.RunOpts{Dir: dstPath}))

			// Wait until status check queue is done, we cannot access the queue's
			// internal information so we rely on the status of the patch being changed.
			assert.Eventually(t, func() bool {
				return unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: normalAGitPR.ID}).Status == issues_model.PullRequestStatusConflict
			}, time.Second*30, time.Millisecond*200)

			test := func(t *testing.T, pr *issues_model.PullRequest) {
				t.Helper()
				if assert.Len(t, pr.ConflictedFiles, 1) {
					assert.Equal(t, "CONTACT", pr.ConflictedFiles[0])
				}
				assert.Equal(t, issues_model.PullRequestStatusConflict, pr.Status)
				assert.Equal(t, 1, pr.CommitsAhead)
				assert.Equal(t, 1, pr.CommitsBehind)
				assert.False(t, pr.Mergeable(t.Context()))
			}

			t.Run("Across repository patch", func(t *testing.T) {
				t.Run("Existing", func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()

					test(t, unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{BaseRepoID: repo.ID, HeadRepoID: forkRepo.ID, HeadBranch: "normal"}, "flow = 0"))
				})

				t.Run("New", func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()

					require.NoError(t, git.NewCommand(t.Context(), "push", "fork", "HEAD:conflict").Run(&git.RunOpts{Dir: dstPath}))
					testPullCreateDirectly(t, session, repo.OwnerName, repo.Name, repo.DefaultBranch, forkRepo.OwnerName, forkRepo.Name, "conflict", "across repo conflict")

					test(t, unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{BaseRepoID: repo.ID, HeadRepoID: forkRepo.ID, HeadBranch: "conflict"}, "flow = 0"))
				})
			})

			t.Run("Same repository patch", func(t *testing.T) {
				t.Run("Existing", func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()

					test(t, unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{BaseRepoID: repo.ID, HeadRepoID: repo.ID, HeadBranch: "normal"}, "flow = 0"))
				})

				t.Run("New", func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()

					require.NoError(t, git.NewCommand(t.Context(), "push", "origin", "HEAD:conflict").Run(&git.RunOpts{Dir: dstPath}))
					testPullCreateDirectly(t, session, repo.OwnerName, repo.Name, repo.DefaultBranch, repo.OwnerName, repo.Name, "conflict", "same repo conflict")

					test(t, unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{BaseRepoID: repo.ID, HeadRepoID: repo.ID, HeadBranch: "conflict"}, "flow = 0"))
				})
			})

			t.Run("AGit", func(t *testing.T) {
				t.Run("Existing", func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()

					test(t, unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{BaseRepoID: repo.ID, HeadRepoID: repo.ID, HeadBranch: "user2/normal", Flow: issues_model.PullRequestFlowAGit}))
				})

				t.Run("New", func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()

					require.NoError(t, git.NewCommand(t.Context(), "push", "origin", "HEAD:refs/for/main", "-o", "topic=conflict").Run(&git.RunOpts{Dir: dstPath}))

					test(t, unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{BaseRepoID: repo.ID, HeadRepoID: repo.ID, HeadBranch: "user2/conflict", Flow: issues_model.PullRequestFlowAGit}))
				})
			})
		})

		// Test that the status is set to empty if the diff is empty.
		t.Run("Empty diff", func(t *testing.T) {
			require.NoError(t, git.NewCommand(t.Context(), "switch", "-c", "empty-patch").AddDynamicArguments(repo.DefaultBranch).Run(&git.RunOpts{Dir: dstPath}))
			require.NoError(t, git.NewCommand(t.Context(), "commit", "--allow-empty", "--message=empty").Run(&git.RunOpts{Dir: dstPath}))

			test := func(t *testing.T, pr *issues_model.PullRequest) {
				t.Helper()

				assert.Empty(t, pr.ConflictedFiles)
				assert.Equal(t, issues_model.PullRequestStatusEmpty, pr.Status)
				assert.Equal(t, 1, pr.CommitsAhead)
				assert.Equal(t, 0, pr.CommitsBehind)
				assert.True(t, pr.Mergeable(t.Context()))
			}

			t.Run("Across repository", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				require.NoError(t, git.NewCommand(t.Context(), "push", "fork", "HEAD:empty-patch").Run(&git.RunOpts{Dir: dstPath}))
				testPullCreateDirectly(t, session, repo.OwnerName, repo.Name, repo.DefaultBranch, forkRepo.OwnerName, forkRepo.Name, "empty-patch", "across repo empty patch")

				test(t, unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{BaseRepoID: repo.ID, HeadRepoID: forkRepo.ID, HeadBranch: "empty-patch"}, "flow = 0"))
			})

			t.Run("Same repository", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				require.NoError(t, git.NewCommand(t.Context(), "push", "origin", "HEAD:empty-patch").Run(&git.RunOpts{Dir: dstPath}))
				testPullCreateDirectly(t, session, repo.OwnerName, repo.Name, repo.DefaultBranch, repo.OwnerName, repo.Name, "empty-patch", "same repo empty patch")

				test(t, unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{BaseRepoID: repo.ID, HeadRepoID: repo.ID, HeadBranch: "empty-patch"}, "flow = 0"))
			})

			t.Run("AGit", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				require.NoError(t, git.NewCommand(t.Context(), "push", "origin", "HEAD:refs/for/main", "-o", "topic=empty-patch").Run(&git.RunOpts{Dir: dstPath}))

				test(t, unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{BaseRepoID: repo.ID, HeadRepoID: repo.ID, HeadBranch: "user2/empty-patch", Flow: issues_model.PullRequestFlowAGit}))
			})
		})

		// If a patch modifies a protected file, it should be marked as such.
		t.Run("Protected file", func(t *testing.T) {
			// Add protected branch.
			link := fmt.Sprintf("/%s/settings/branches/edit", repo.FullName())
			session.MakeRequest(t, NewRequestWithValues(t, "POST", link, map[string]string{
				"_csrf":                   GetCSRF(t, session, link),
				"rule_name":               "main",
				"protected_file_patterns": "LICENSE",
			}), http.StatusSeeOther)

			require.NoError(t, git.NewCommand(t.Context(), "switch", "-c", "protected").AddDynamicArguments(repo.DefaultBranch).Run(&git.RunOpts{Dir: dstPath}))
			require.NoError(t, os.WriteFile(filepath.Join(dstPath, "LICENSE"), []byte(`# "THE SPEZI-WARE LICENSE" (Revision 2137):

As long as you retain this notice, you can do whatever you want with this
project. If we meet some day, and you think this stuff is worth it, you
can buy me/us a Paulaner Spezi in return.        ~sdomi, Project SERVFAIL`), 0o600))
			require.NoError(t, git.NewCommand(t.Context(), "add", "LICENSE").Run(&git.RunOpts{Dir: dstPath}))
			require.NoError(t, git.NewCommand(t.Context(), "commit", "--message=servfail").Run(&git.RunOpts{Dir: dstPath}))

			test := func(t *testing.T, pr *issues_model.PullRequest) {
				t.Helper()
				if assert.Len(t, pr.ChangedProtectedFiles, 1) {
					assert.Equal(t, "license", pr.ChangedProtectedFiles[0])
				}
				assert.Equal(t, issues_model.PullRequestStatusMergeable, pr.Status)
				assert.Equal(t, 1, pr.CommitsAhead)
				assert.Equal(t, 0, pr.CommitsBehind)
				assert.True(t, pr.Mergeable(t.Context()))
			}

			t.Run("Across repository", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				require.NoError(t, git.NewCommand(t.Context(), "push", "fork", "HEAD:protected").Run(&git.RunOpts{Dir: dstPath}))
				testPullCreateDirectly(t, session, repo.OwnerName, repo.Name, repo.DefaultBranch, forkRepo.OwnerName, forkRepo.Name, "protected", "accros repo protected")

				test(t, unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{BaseRepoID: repo.ID, HeadRepoID: forkRepo.ID, HeadBranch: "protected"}, "flow = 0"))
			})

			t.Run("Same repository", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				require.NoError(t, git.NewCommand(t.Context(), "push", "origin", "HEAD:protected").Run(&git.RunOpts{Dir: dstPath}))
				testPullCreateDirectly(t, session, repo.OwnerName, repo.Name, repo.DefaultBranch, repo.OwnerName, repo.Name, "protected", "same repo protected")

				test(t, unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{BaseRepoID: repo.ID, HeadRepoID: repo.ID, HeadBranch: "protected"}, "flow = 0"))
			})

			t.Run("AGit", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				require.NoError(t, git.NewCommand(t.Context(), "push", "origin", "HEAD:refs/for/main", "-o", "topic=protected").Run(&git.RunOpts{Dir: dstPath}))

				test(t, unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{BaseRepoID: repo.ID, HeadRepoID: repo.ID, HeadBranch: "user2/protected", Flow: issues_model.PullRequestFlowAGit}))
			})
		})

		// If the head branch is a ancestor of the base branch, then it should be marked.
		t.Run("Ancestor", func(t *testing.T) {
			require.NoError(t, git.NewCommand(t.Context(), "push", "origin", "protected:protected").Run(&git.RunOpts{Dir: dstPath}))
			require.NoError(t, git.NewCommand(t.Context(), "switch").AddDynamicArguments(repo.DefaultBranch).Run(&git.RunOpts{Dir: dstPath}))

			test := func(t *testing.T, pr *issues_model.PullRequest) {
				t.Helper()
				assert.Equal(t, issues_model.PullRequestStatusAncestor, pr.Status)
				assert.Equal(t, 0, pr.CommitsAhead)
				assert.Equal(t, 1, pr.CommitsBehind)
				assert.True(t, pr.Mergeable(t.Context()))
			}

			// AGit has a check to not allow AGit to get in this state.

			t.Run("Across repository", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				require.NoError(t, git.NewCommand(t.Context(), "push", "fork", "HEAD:ancestor").Run(&git.RunOpts{Dir: dstPath}))
				testPullCreateDirectly(t, session, repo.OwnerName, repo.Name, "protected", forkRepo.OwnerName, forkRepo.Name, "ancestor", "accros repo ancestor")

				test(t, unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{BaseRepoID: repo.ID, HeadRepoID: forkRepo.ID, HeadBranch: "ancestor"}, "flow = 0"))
			})

			t.Run("Same repository", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				require.NoError(t, git.NewCommand(t.Context(), "push", "origin", "HEAD:ancestor").Run(&git.RunOpts{Dir: dstPath}))
				testPullCreateDirectly(t, session, repo.OwnerName, repo.Name, "protected", repo.OwnerName, repo.Name, "ancestor", "same repo ancestor")

				test(t, unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{BaseRepoID: repo.ID, HeadRepoID: repo.ID, HeadBranch: "ancestor"}, "flow = 0"))
			})
		})
	})
}
