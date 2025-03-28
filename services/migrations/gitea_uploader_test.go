// Copyright 2019 The Gitea Authors. All rights reserved.
// Copyright 2018 Jonas Franz. All rights reserved.
// SPDX-License-Identifier: MIT

package migrations

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"forgejo.org/models/db"
	issues_model "forgejo.org/models/issues"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/git"
	"forgejo.org/modules/gitrepo"
	"forgejo.org/modules/graceful"
	"forgejo.org/modules/log"
	base "forgejo.org/modules/migration"
	"forgejo.org/modules/optional"
	"forgejo.org/modules/structs"
	"forgejo.org/modules/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGiteaUploadRepo(t *testing.T) {
	// FIXME: Since no accesskey or user/password will trigger rate limit of github, just skip
	t.Skip()

	unittest.PrepareTestEnv(t)

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})

	var (
		ctx        = t.Context()
		downloader = NewGithubDownloaderV3(ctx, "https://github.com", "", "", "", "go-xorm", "builder")
		repoName   = "builder-" + time.Now().Format("2006-01-02-15-04-05")
		uploader   = NewGiteaLocalUploader(graceful.GetManager().HammerContext(), user, user.Name, repoName)
	)

	err := migrateRepository(db.DefaultContext, user, downloader, uploader, base.MigrateOptions{
		CloneAddr:    "https://github.com/go-xorm/builder",
		RepoName:     repoName,
		AuthUsername: "",

		Wiki:         true,
		Issues:       true,
		Milestones:   true,
		Labels:       true,
		Releases:     true,
		Comments:     true,
		PullRequests: true,
		Private:      true,
		Mirror:       false,
	}, nil)
	require.NoError(t, err)

	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{OwnerID: user.ID, Name: repoName})
	assert.True(t, repo.HasWiki())
	assert.Equal(t, repo_model.RepositoryReady, repo.Status)

	milestones, err := db.Find[issues_model.Milestone](db.DefaultContext, issues_model.FindMilestoneOptions{
		RepoID:   repo.ID,
		IsClosed: optional.Some(false),
	})
	require.NoError(t, err)
	assert.Len(t, milestones, 1)

	milestones, err = db.Find[issues_model.Milestone](db.DefaultContext, issues_model.FindMilestoneOptions{
		RepoID:   repo.ID,
		IsClosed: optional.Some(true),
	})
	require.NoError(t, err)
	assert.Empty(t, milestones)

	labels, err := issues_model.GetLabelsByRepoID(ctx, repo.ID, "", db.ListOptions{})
	require.NoError(t, err)
	assert.Len(t, labels, 12)

	releases, err := db.Find[repo_model.Release](db.DefaultContext, repo_model.FindReleasesOptions{
		ListOptions: db.ListOptions{
			PageSize: 10,
			Page:     0,
		},
		IncludeTags: true,
		RepoID:      repo.ID,
	})
	require.NoError(t, err)
	assert.Len(t, releases, 8)

	releases, err = db.Find[repo_model.Release](db.DefaultContext, repo_model.FindReleasesOptions{
		ListOptions: db.ListOptions{
			PageSize: 10,
			Page:     0,
		},
		IncludeTags: false,
		RepoID:      repo.ID,
	})
	require.NoError(t, err)
	assert.Len(t, releases, 1)

	issues, err := issues_model.Issues(db.DefaultContext, &issues_model.IssuesOptions{
		RepoIDs:  []int64{repo.ID},
		IsPull:   optional.Some(false),
		SortType: "oldest",
	})
	require.NoError(t, err)
	assert.Len(t, issues, 15)
	require.NoError(t, issues[0].LoadDiscussComments(db.DefaultContext))
	assert.Empty(t, issues[0].Comments)

	pulls, _, err := issues_model.PullRequests(db.DefaultContext, repo.ID, &issues_model.PullRequestsOptions{
		SortType: "oldest",
	})
	require.NoError(t, err)
	assert.Len(t, pulls, 30)
	require.NoError(t, pulls[0].LoadIssue(db.DefaultContext))
	require.NoError(t, pulls[0].Issue.LoadDiscussComments(db.DefaultContext))
	assert.Len(t, pulls[0].Issue.Comments, 2)
}

func TestGiteaUploadRemapLocalUser(t *testing.T) {
	unittest.PrepareTestEnv(t)
	doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	repoName := "migrated"
	uploader := NewGiteaLocalUploader(t.Context(), doer, doer.Name, repoName)
	// call remapLocalUser
	uploader.sameApp = true

	externalID := int64(1234567)
	externalName := "username"
	source := base.Release{
		PublisherID:   externalID,
		PublisherName: externalName,
	}

	//
	// The externalID does not match any existing user, everything
	// belongs to the Ghost user
	//
	target := repo_model.Release{}
	uploader.userMap = make(map[int64]int64)
	err := uploader.remapUser(&source, &target)
	require.NoError(t, err)
	assert.EqualValues(t, user_model.GhostUserID, target.GetUserID())

	//
	// The externalID matches a known user but the name does not match,
	// everything belongs to the Ghost user
	//
	source.PublisherID = user.ID
	target = repo_model.Release{}
	uploader.userMap = make(map[int64]int64)
	err = uploader.remapUser(&source, &target)
	require.NoError(t, err)
	assert.EqualValues(t, user_model.GhostUserID, target.GetUserID())

	//
	// The externalID and externalName match an existing user, everything
	// belongs to the existing user
	//
	source.PublisherName = user.Name
	target = repo_model.Release{}
	uploader.userMap = make(map[int64]int64)
	err = uploader.remapUser(&source, &target)
	require.NoError(t, err)
	assert.Equal(t, user.ID, target.GetUserID())
}

func TestGiteaUploadRemapExternalUser(t *testing.T) {
	unittest.PrepareTestEnv(t)
	doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})

	repoName := "migrated"
	uploader := NewGiteaLocalUploader(t.Context(), doer, doer.Name, repoName)
	uploader.gitServiceType = structs.GiteaService
	// call remapExternalUser
	uploader.sameApp = false

	externalID := int64(1234567)
	externalName := "username"
	source := base.Release{
		PublisherID:   externalID,
		PublisherName: externalName,
	}

	//
	// When there is no user linked to the external ID, the migrated data is authored
	// by the Ghost user
	//
	uploader.userMap = make(map[int64]int64)
	target := repo_model.Release{}
	err := uploader.remapUser(&source, &target)
	require.NoError(t, err)
	assert.EqualValues(t, user_model.GhostUserID, target.GetUserID())

	//
	// Link the external ID to an existing user
	//
	linkedUser := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	externalLoginUser := &user_model.ExternalLoginUser{
		ExternalID:    strconv.FormatInt(externalID, 10),
		UserID:        linkedUser.ID,
		LoginSourceID: 0,
		Provider:      structs.GiteaService.Name(),
	}
	err = user_model.LinkExternalToUser(db.DefaultContext, linkedUser, externalLoginUser)
	require.NoError(t, err)

	//
	// When a user is linked to the external ID, it becomes the author of
	// the migrated data
	//
	uploader.userMap = make(map[int64]int64)
	target = repo_model.Release{}
	err = uploader.remapUser(&source, &target)
	require.NoError(t, err)
	assert.Equal(t, linkedUser.ID, target.GetUserID())
}

func TestGiteaUploadUpdateGitForPullRequest(t *testing.T) {
	unittest.PrepareTestEnv(t)

	//
	// fromRepo master
	//
	fromRepo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	baseRef := "master"
	require.NoError(t, git.InitRepository(git.DefaultContext, fromRepo.RepoPath(), false, fromRepo.ObjectFormatName))
	err := git.NewCommand(git.DefaultContext, "symbolic-ref").AddDynamicArguments("HEAD", git.BranchPrefix+baseRef).Run(&git.RunOpts{Dir: fromRepo.RepoPath()})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(fromRepo.RepoPath(), "README.md"), []byte(fmt.Sprintf("# Testing Repository\n\nOriginally created in: %s", fromRepo.RepoPath())), 0o644))
	require.NoError(t, git.AddChanges(fromRepo.RepoPath(), true))
	signature := git.Signature{
		Email: "test@example.com",
		Name:  "test",
		When:  time.Now(),
	}
	require.NoError(t, git.CommitChanges(fromRepo.RepoPath(), git.CommitChangesOptions{
		Committer: &signature,
		Author:    &signature,
		Message:   "Initial Commit",
	}))
	fromGitRepo, err := gitrepo.OpenRepository(git.DefaultContext, fromRepo)
	require.NoError(t, err)
	defer fromGitRepo.Close()
	baseSHA, err := fromGitRepo.GetBranchCommitID(baseRef)
	require.NoError(t, err)

	//
	// fromRepo branch1
	//
	headRef := "branch1"
	_, _, err = git.NewCommand(git.DefaultContext, "checkout", "-b").AddDynamicArguments(headRef).RunStdString(&git.RunOpts{Dir: fromRepo.RepoPath()})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(fromRepo.RepoPath(), "README.md"), []byte("SOMETHING"), 0o644))
	require.NoError(t, git.AddChanges(fromRepo.RepoPath(), true))
	signature.When = time.Now()
	require.NoError(t, git.CommitChanges(fromRepo.RepoPath(), git.CommitChangesOptions{
		Committer: &signature,
		Author:    &signature,
		Message:   "Pull request",
	}))
	require.NoError(t, err)
	headSHA, err := fromGitRepo.GetBranchCommitID(headRef)
	require.NoError(t, err)

	fromRepoOwner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: fromRepo.OwnerID})

	//
	// forkRepo branch2
	//
	forkHeadRef := "branch2"
	forkRepo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 8})
	require.NoError(t, git.CloneWithArgs(git.DefaultContext, nil, fromRepo.RepoPath(), forkRepo.RepoPath(), git.CloneRepoOptions{
		Branch: headRef,
	}))
	_, _, err = git.NewCommand(git.DefaultContext, "checkout", "-b").AddDynamicArguments(forkHeadRef).RunStdString(&git.RunOpts{Dir: forkRepo.RepoPath()})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(forkRepo.RepoPath(), "README.md"), []byte(fmt.Sprintf("# branch2 %s", forkRepo.RepoPath())), 0o644))
	require.NoError(t, git.AddChanges(forkRepo.RepoPath(), true))
	require.NoError(t, git.CommitChanges(forkRepo.RepoPath(), git.CommitChangesOptions{
		Committer: &signature,
		Author:    &signature,
		Message:   "branch2 commit",
	}))
	forkGitRepo, err := gitrepo.OpenRepository(git.DefaultContext, forkRepo)
	require.NoError(t, err)
	defer forkGitRepo.Close()
	forkHeadSHA, err := forkGitRepo.GetBranchCommitID(forkHeadRef)
	require.NoError(t, err)

	toRepoName := "migrated"
	uploader := NewGiteaLocalUploader(t.Context(), fromRepoOwner, fromRepoOwner.Name, toRepoName)
	uploader.gitServiceType = structs.GiteaService
	require.NoError(t, uploader.CreateRepo(&base.Repository{
		Description: "description",
		OriginalURL: fromRepo.RepoPath(),
		CloneURL:    fromRepo.RepoPath(),
		IsPrivate:   false,
		IsMirror:    true,
	}, base.MigrateOptions{
		GitServiceType: structs.GiteaService,
		Private:        false,
		Mirror:         true,
	}))

	for _, testCase := range []struct {
		name        string
		head        string
		logFilter   []string
		logFiltered []bool
		pr          base.PullRequest
	}{
		{
			name: "fork, good Head.SHA",
			head: fmt.Sprintf("%s/%s", forkRepo.OwnerName, forkHeadRef),
			pr: base.PullRequest{
				PatchURL: "",
				Number:   1,
				State:    "open",
				Base: base.PullRequestBranch{
					CloneURL:  fromRepo.RepoPath(),
					Ref:       baseRef,
					SHA:       baseSHA,
					RepoName:  fromRepo.Name,
					OwnerName: fromRepo.OwnerName,
				},
				Head: base.PullRequestBranch{
					CloneURL:  forkRepo.RepoPath(),
					Ref:       forkHeadRef,
					SHA:       forkHeadSHA,
					RepoName:  forkRepo.Name,
					OwnerName: forkRepo.OwnerName,
				},
			},
		},
		{
			name: "fork, invalid Head.Ref",
			head: "unknown repository",
			pr: base.PullRequest{
				PatchURL: "",
				Number:   1,
				State:    "open",
				Base: base.PullRequestBranch{
					CloneURL:  fromRepo.RepoPath(),
					Ref:       baseRef,
					SHA:       baseSHA,
					RepoName:  fromRepo.Name,
					OwnerName: fromRepo.OwnerName,
				},
				Head: base.PullRequestBranch{
					CloneURL:  forkRepo.RepoPath(),
					Ref:       "INVALID",
					SHA:       forkHeadSHA,
					RepoName:  forkRepo.Name,
					OwnerName: forkRepo.OwnerName,
				},
			},
			logFilter:   []string{"Fetch branch from"},
			logFiltered: []bool{true},
		},
		{
			name: "invalid fork CloneURL",
			head: "unknown repository",
			pr: base.PullRequest{
				PatchURL: "",
				Number:   1,
				State:    "open",
				Base: base.PullRequestBranch{
					CloneURL:  fromRepo.RepoPath(),
					Ref:       baseRef,
					SHA:       baseSHA,
					RepoName:  fromRepo.Name,
					OwnerName: fromRepo.OwnerName,
				},
				Head: base.PullRequestBranch{
					CloneURL:  "UNLIKELY",
					Ref:       forkHeadRef,
					SHA:       forkHeadSHA,
					RepoName:  forkRepo.Name,
					OwnerName: "WRONG",
				},
			},
			logFilter:   []string{"AddRemote"},
			logFiltered: []bool{true},
		},
		{
			name: "no fork, good Head.SHA",
			head: headRef,
			pr: base.PullRequest{
				PatchURL: "",
				Number:   1,
				State:    "open",
				Base: base.PullRequestBranch{
					CloneURL:  fromRepo.RepoPath(),
					Ref:       baseRef,
					SHA:       baseSHA,
					RepoName:  fromRepo.Name,
					OwnerName: fromRepo.OwnerName,
				},
				Head: base.PullRequestBranch{
					CloneURL:  fromRepo.RepoPath(),
					Ref:       headRef,
					SHA:       headSHA,
					RepoName:  fromRepo.Name,
					OwnerName: fromRepo.OwnerName,
				},
			},
		},
		{
			name: "no fork, empty Head.SHA",
			head: headRef,
			pr: base.PullRequest{
				PatchURL: "",
				Number:   1,
				State:    "open",
				Base: base.PullRequestBranch{
					CloneURL:  fromRepo.RepoPath(),
					Ref:       baseRef,
					SHA:       baseSHA,
					RepoName:  fromRepo.Name,
					OwnerName: fromRepo.OwnerName,
				},
				Head: base.PullRequestBranch{
					CloneURL:  fromRepo.RepoPath(),
					Ref:       headRef,
					SHA:       "",
					RepoName:  fromRepo.Name,
					OwnerName: fromRepo.OwnerName,
				},
			},
			logFilter:   []string{"Empty reference", "Cannot remove local head"},
			logFiltered: []bool{true, false},
		},
		{
			name: "no fork, invalid Head.SHA",
			head: headRef,
			pr: base.PullRequest{
				PatchURL: "",
				Number:   1,
				State:    "open",
				Base: base.PullRequestBranch{
					CloneURL:  fromRepo.RepoPath(),
					Ref:       baseRef,
					SHA:       baseSHA,
					RepoName:  fromRepo.Name,
					OwnerName: fromRepo.OwnerName,
				},
				Head: base.PullRequestBranch{
					CloneURL:  fromRepo.RepoPath(),
					Ref:       headRef,
					SHA:       "brokenSHA",
					RepoName:  fromRepo.Name,
					OwnerName: fromRepo.OwnerName,
				},
			},
			logFilter:   []string{"Deprecated local head"},
			logFiltered: []bool{true},
		},
		{
			name: "no fork, not found Head.SHA",
			head: headRef,
			pr: base.PullRequest{
				PatchURL: "",
				Number:   1,
				State:    "open",
				Base: base.PullRequestBranch{
					CloneURL:  fromRepo.RepoPath(),
					Ref:       baseRef,
					SHA:       baseSHA,
					RepoName:  fromRepo.Name,
					OwnerName: fromRepo.OwnerName,
				},
				Head: base.PullRequestBranch{
					CloneURL:  fromRepo.RepoPath(),
					Ref:       headRef,
					SHA:       "2697b352310fcd01cbd1f3dbd43b894080027f68",
					RepoName:  fromRepo.Name,
					OwnerName: fromRepo.OwnerName,
				},
			},
			logFilter:   []string{"Deprecated local head", "Cannot remove local head"},
			logFiltered: []bool{true, false},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			stopMark := fmt.Sprintf(">>>>>>>>>>>>>STOP: %s<<<<<<<<<<<<<<<", testCase.name)

			logChecker, cleanup := test.NewLogChecker(log.DEFAULT, log.INFO)
			logChecker.Filter(testCase.logFilter...).StopMark(stopMark)
			defer cleanup()

			testCase.pr.EnsuredSafe = true

			head, err := uploader.updateGitForPullRequest(&testCase.pr)
			require.NoError(t, err)
			assert.Equal(t, testCase.head, head)

			log.Info(stopMark)

			logFiltered, logStopped := logChecker.Check(5 * time.Second)
			assert.True(t, logStopped)
			if len(testCase.logFilter) > 0 {
				assert.Equal(t, testCase.logFiltered, logFiltered, "for log message filters: %v", testCase.logFilter)
			}
		})
	}
}
