// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	actions_model "forgejo.org/models/actions"
	auth_model "forgejo.org/models/auth"
	"forgejo.org/models/db"
	git_model "forgejo.org/models/git"
	issues_model "forgejo.org/models/issues"
	repo_model "forgejo.org/models/repo"
	unit_model "forgejo.org/models/unit"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	actions_module "forgejo.org/modules/actions"
	"forgejo.org/modules/git"
	"forgejo.org/modules/gitrepo"
	"forgejo.org/modules/setting"
	api "forgejo.org/modules/structs"
	"forgejo.org/modules/test"
	webhook_module "forgejo.org/modules/webhook"
	actions_service "forgejo.org/services/actions"
	issue_service "forgejo.org/services/issue"
	pull_service "forgejo.org/services/pull"
	release_service "forgejo.org/services/release"
	repo_service "forgejo.org/services/repository"
	files_service "forgejo.org/services/repository/files"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPullRequestCommitStatus(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2}) // owner of the base repo
		session := loginUser(t, "user2")
		token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteIssue)

		// prepare the repository
		files := make([]*files_service.ChangeRepoFile, 0, 10)
		for _, onType := range []string{
			"opened",
			"synchronize",
			"labeled",
			"unlabeled",
			"assigned",
			"unassigned",
			"milestoned",
			"demilestoned",
			"closed",
			"reopened",
		} {
			files = append(files, &files_service.ChangeRepoFile{
				Operation: "create",
				TreePath:  fmt.Sprintf(".forgejo/workflows/%s.yml", onType),
				ContentReader: strings.NewReader(fmt.Sprintf(`name: %[1]s
on:
  pull_request:
    types:
      - %[1]s
jobs:
  %[1]s:
    runs-on: docker
    steps:
      - run: true
`, onType)),
			})
		}
		baseRepo, _, f := tests.CreateDeclarativeRepo(t, user2, "repo-pull-request",
			[]unit_model.Type{unit_model.TypeActions}, nil, files)
		defer f()
		baseGitRepo, err := gitrepo.OpenRepository(db.DefaultContext, baseRepo)
		require.NoError(t, err)
		defer func() {
			baseGitRepo.Close()
		}()

		// prepare the repository labels
		labelStr := "/api/v1/repos/user2/repo-pull-request/labels"
		labelsCount := 2
		labels := make([]*api.Label, labelsCount)
		for i := 0; i < labelsCount; i++ {
			color := "abcdef"
			req := NewRequestWithJSON(t, "POST", labelStr, &api.CreateLabelOption{
				Name:  fmt.Sprintf("label%d", i),
				Color: color,
			}).AddTokenAuth(token)
			resp := MakeRequest(t, req, http.StatusCreated)
			labels[i] = new(api.Label)
			DecodeJSON(t, resp, &labels[i])
			assert.Equal(t, color, labels[i].Color)
		}

		// create the pull request
		testEditFileToNewBranch(t, session, "user2", "repo-pull-request", "main", "wip-something", "README.md", "Hello, world 1")
		testPullCreate(t, session, "user2", "repo-pull-request", true, "main", "wip-something", "Commit status PR")
		pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{BaseRepoID: baseRepo.ID})
		require.NoError(t, pr.LoadIssue(db.DefaultContext))

		// prepare the assignees
		issueURL := fmt.Sprintf("/api/v1/repos/%s/%s/issues/%s", "user2", "repo-pull-request", fmt.Sprintf("%d", pr.Issue.Index))

		// prepare the labels
		labelURL := fmt.Sprintf("%s/labels", issueURL)

		// prepare the milestone
		milestoneStr := "/api/v1/repos/user2/repo-pull-request/milestones"
		req := NewRequestWithJSON(t, "POST", milestoneStr, &api.CreateMilestoneOption{
			Title: "mymilestone",
			State: "open",
		}).AddTokenAuth(token)
		resp := MakeRequest(t, req, http.StatusCreated)
		milestone := new(api.Milestone)
		DecodeJSON(t, resp, &milestone)

		// check that one of the status associated with the commit sha matches both
		// context & state
		checkCommitStatus := func(sha, context string, state api.CommitStatusState) bool {
			commitStatuses, _, err := git_model.GetLatestCommitStatus(db.DefaultContext, pr.BaseRepoID, sha, db.ListOptionsAll)
			require.NoError(t, err)
			for _, commitStatus := range commitStatuses {
				if state == commitStatus.State && context == commitStatus.Context {
					return true
				}
			}
			return false
		}

		assertActionRun := func(t *testing.T, sha, onType string, action api.HookIssueAction, actionRun *actions_model.ActionRun) {
			assert.Equal(t, fmt.Sprintf("%s.yml", onType), actionRun.WorkflowID)
			assert.Equal(t, sha, actionRun.CommitSHA)
			assert.Equal(t, actions_module.GithubEventPullRequest, actionRun.TriggerEvent)
			event, err := actionRun.GetPullRequestEventPayload()
			require.NoError(t, err)
			assert.Equal(t, action, event.Action)
		}

		type assertType func(t *testing.T, sha, onType string, action api.HookIssueAction, actionRuns []*actions_model.ActionRun)
		assertActionRuns := func(t *testing.T, sha, onType string, action api.HookIssueAction, actionRuns []*actions_model.ActionRun) {
			require.Len(t, actionRuns, 1)
			assertActionRun(t, sha, onType, action, actionRuns[0])
		}

		for _, testCase := range []struct {
			onType         string
			jobID          string
			doSomething    func()
			actionRunCount int
			action         api.HookIssueAction
			assert         assertType
		}{
			{
				onType:         "opened",
				doSomething:    func() {},
				actionRunCount: 1,
				action:         api.HookIssueOpened,
				assert:         assertActionRuns,
			},
			{
				onType: "synchronize",
				doSomething: func() {
					testEditFile(t, session, "user2", "repo-pull-request", "wip-something", "README.md", "Hello, world 2")
				},
				actionRunCount: 1,
				action:         api.HookIssueSynchronized,
				assert:         assertActionRuns,
			},
			{
				onType: "labeled",
				doSomething: func() {
					req := NewRequestWithJSON(t, "POST", labelURL, &api.IssueLabelsOption{
						Labels: []any{labels[0].ID, labels[1].ID},
					}).AddTokenAuth(token)
					MakeRequest(t, req, http.StatusOK)
				},
				actionRunCount: 2,
				action:         api.HookIssueLabelUpdated,
				assert: func(t *testing.T, sha, onType string, action api.HookIssueAction, actionRuns []*actions_model.ActionRun) {
					assertActionRun(t, sha, onType, api.HookIssueLabelUpdated, actionRuns[0])
					assertActionRun(t, sha, onType, api.HookIssueLabelUpdated, actionRuns[1])
				},
			},
			{
				onType: "unlabeled",
				doSomething: func() {
					req := NewRequestWithJSON(t, "PUT", labelURL, &api.IssueLabelsOption{
						Labels: []any{labels[0].ID},
					}).AddTokenAuth(token)
					MakeRequest(t, req, http.StatusOK)
				},
				actionRunCount: 3,
				action:         api.HookIssueLabelCleared,
				assert: func(t *testing.T, sha, onType string, action api.HookIssueAction, actionRuns []*actions_model.ActionRun) {
					foundPayloadWithLabels := false
					knownLabels := []string{"label0", "label1"}
					for _, actionRun := range actionRuns {
						assert.Equal(t, sha, actionRun.CommitSHA)
						assert.Equal(t, actions_module.GithubEventPullRequest, actionRun.TriggerEvent)
						event, err := actionRun.GetPullRequestEventPayload()
						require.NoError(t, err)
						switch event.Action {
						case api.HookIssueLabelUpdated:
							assert.Equal(t, "labeled.yml", actionRun.WorkflowID)
							assert.Equal(t, "label0", event.Label.Name)
							require.Len(t, event.PullRequest.Labels, 1)
							assert.Contains(t, "label0", event.PullRequest.Labels[0].Name)
						case api.HookIssueLabelCleared:
							assert.Equal(t, "unlabeled.yml", actionRun.WorkflowID)
							assert.Contains(t, knownLabels, event.Label.Name)
							if len(event.PullRequest.Labels) > 0 {
								foundPayloadWithLabels = true
								assert.Contains(t, knownLabels, event.PullRequest.Labels[0].Name)
							}
						default:
							require.Fail(t, fmt.Sprintf("unexpected action '%s'", event.Action))
						}
					}
					assert.True(t, foundPayloadWithLabels, "expected at least one clear label payload with non empty labels")
				},
			},
			{
				onType: "assigned",
				doSomething: func() {
					req := NewRequestWithJSON(t, "PATCH", issueURL, &api.EditIssueOption{
						Assignees: []string{"user2"},
					}).AddTokenAuth(token)
					MakeRequest(t, req, http.StatusCreated)
				},
				actionRunCount: 1,
				action:         api.HookIssueAssigned,
				assert:         assertActionRuns,
			},
			{
				onType: "unassigned",
				doSomething: func() {
					req := NewRequestWithJSON(t, "PATCH", issueURL, &api.EditIssueOption{
						Assignees: []string{},
					}).AddTokenAuth(token)
					MakeRequest(t, req, http.StatusCreated)
				},
				actionRunCount: 1,
				action:         api.HookIssueUnassigned,
				assert:         assertActionRuns,
			},
			{
				onType: "milestoned",
				doSomething: func() {
					req := NewRequestWithJSON(t, "PATCH", issueURL, &api.EditIssueOption{
						Milestone: &milestone.ID,
					}).AddTokenAuth(token)
					MakeRequest(t, req, http.StatusCreated)
				},
				actionRunCount: 1,
				action:         api.HookIssueMilestoned,
				assert:         assertActionRuns,
			},
			{
				onType: "demilestoned",
				doSomething: func() {
					var zero int64
					req := NewRequestWithJSON(t, "PATCH", issueURL, &api.EditIssueOption{
						Milestone: &zero,
					}).AddTokenAuth(token)
					MakeRequest(t, req, http.StatusCreated)
				},
				actionRunCount: 1,
				action:         api.HookIssueDemilestoned,
				assert:         assertActionRuns,
			},
			{
				onType: "closed",
				doSomething: func() {
					sha, err := baseGitRepo.GetRefCommitID(pr.GetGitRefName())
					require.NoError(t, err)
					err = issue_service.ChangeStatus(db.DefaultContext, pr.Issue, user2, sha, true)
					require.NoError(t, err)
				},
				actionRunCount: 1,
				action:         api.HookIssueClosed,
				assert:         assertActionRuns,
			},
			{
				onType: "reopened",
				doSomething: func() {
					sha, err := baseGitRepo.GetRefCommitID(pr.GetGitRefName())
					require.NoError(t, err)
					err = issue_service.ChangeStatus(db.DefaultContext, pr.Issue, user2, sha, false)
					require.NoError(t, err)
				},
				actionRunCount: 1,
				action:         api.HookIssueReOpened,
				assert:         assertActionRuns,
			},
		} {
			t.Run(testCase.onType, func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()
				defer func() {
					// cleanup leftovers, start from scratch
					unittest.AssertSuccessfulDelete(t, &actions_model.ActionRun{RepoID: baseRepo.ID})
					unittest.AssertSuccessfulDelete(t, &actions_model.ActionRunJob{RepoID: baseRepo.ID})
				}()

				// trigger the onType event
				testCase.doSomething()
				count := testCase.actionRunCount
				context := fmt.Sprintf("%[1]s / %[1]s (pull_request)", testCase.onType)

				var actionRuns []*actions_model.ActionRun

				// wait for ActionRun(s) to be created
				require.Eventually(t, func() bool {
					actionRuns = make([]*actions_model.ActionRun, 0)
					require.NoError(t, db.GetEngine(db.DefaultContext).Where("repo_id=?", baseRepo.ID).Find(&actionRuns))
					return len(actionRuns) == count
				}, 30*time.Second, 1*time.Second)

				// verify the expected  ActionRuns were created
				sha, err := baseGitRepo.GetRefCommitID(pr.GetGitRefName())
				require.NoError(t, err)
				// verify the commit status changes to CommitStatusSuccess when the job changes to StatusSuccess
				require.Eventually(t, func() bool {
					return checkCommitStatus(sha, context, api.CommitStatusPending)
				}, 30*time.Second, 1*time.Second)
				for _, actionRun := range actionRuns {
					// verify the expected  ActionRunJob was created and is StatusWaiting
					job := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRunJob{RunID: actionRun.ID, CommitSHA: sha})
					assert.Equal(t, actions_model.StatusWaiting, job.Status)

					// change the state of the job to success
					job.Status = actions_model.StatusSuccess
					actions_service.CreateCommitStatus(db.DefaultContext, job)
				}
				// verify the commit status changed to CommitStatusSuccess because the job(s) changed to StatusSuccess
				require.Eventually(t, func() bool {
					return checkCommitStatus(sha, context, api.CommitStatusSuccess)
				}, 30*time.Second, 1*time.Second)

				testCase.assert(t, sha, testCase.onType, testCase.action, actionRuns)
			})
		}
	})
}

func TestPullRequestWithInvalidWorkflow(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2}) // owner of the base repo
		session := loginUser(t, "user2")

		// prepare the repository
		baseRepo, _, f := tests.CreateDeclarativeRepo(t, user2, "repo-pull-request",
			[]unit_model.Type{unit_model.TypeActions}, nil, []*files_service.ChangeRepoFile{
				{
					Operation: "create",
					TreePath:  ".forgejo/workflows/broken.yml",
					ContentReader: strings.NewReader(`name: broken
on:
pull_request:
types:
	- opened
jobs:
test:
runs-on: docker
	- run: true
`),
				},
			})
		defer f()
		baseGitRepo, err := gitrepo.OpenRepository(t.Context(), baseRepo)
		require.NoError(t, err)
		defer func() {
			baseGitRepo.Close()
		}()

		// create the pull request
		testEditFileToNewBranch(t, session, "user2", "repo-pull-request", "main", "wip-something", "README.md", "Hello, world 1")
		testPullCreate(t, session, "user2", "repo-pull-request", true, "main", "wip-something", "Commit status PR")
		pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{BaseRepoID: baseRepo.ID})
		require.NoError(t, pr.LoadIssue(t.Context()))

		// check that one of the status associated with the commit sha matches both
		// context & state
		checkCommitStatus := func(sha, context string, state api.CommitStatusState) bool {
			commitStatuses, _, err := git_model.GetLatestCommitStatus(t.Context(), pr.BaseRepoID, sha, db.ListOptionsAll)
			require.NoError(t, err)
			for _, commitStatus := range commitStatuses {
				if state == commitStatus.State && context == commitStatus.Context {
					return true
				}
			}
			return false
		}

		var actionRuns []*actions_model.ActionRun

		// wait for ActionRun(s) to be created
		require.Eventually(t, func() bool {
			actionRuns = make([]*actions_model.ActionRun, 0)
			require.NoError(t, db.GetEngine(t.Context()).Where("event=? AND status=? AND repo_id=?", "pull_request", actions_model.StatusFailure, baseRepo.ID).Find(&actionRuns))
			return len(actionRuns) == 1
		}, 30*time.Second, 1*time.Second)

		// verify the expected  ActionRuns were created
		sha, err := baseGitRepo.GetRefCommitID(pr.GetGitRefName())
		require.NoError(t, err)

		// verify the commit status changes to CommitStatusFailure
		require.Eventually(t, func() bool {
			return checkCommitStatus(sha, "broken.yml / Update README.md (pull_request)", api.CommitStatusFailure)
		}, 30*time.Second, 1*time.Second)

		require.Len(t, actionRuns, 1)
		actionRun := actionRuns[0]
		// verify the expected  ActionRunJob was created and is StatusFailure
		job := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRunJob{RunID: actionRun.ID, CommitSHA: sha})
		assert.Equal(t, actions_model.StatusFailure, job.Status)
		assert.Equal(t, "broken.yml", actionRun.WorkflowID)
		assert.Equal(t, sha, actionRun.CommitSHA)
		assert.Equal(t, actions_module.GithubEventPullRequest, actionRun.TriggerEvent)
		event, err := actionRun.GetPullRequestEventPayload()
		require.NoError(t, err)
		assert.Equal(t, api.HookIssueOpened, event.Action)
	})
}

func TestPullRequestTargetEvent(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2}) // owner of the base repo
		org3 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 3})  // owner of the forked repo

		// create the base repo
		baseRepo, _, f := tests.CreateDeclarativeRepo(t, user2, "repo-pull-request-target",
			[]unit_model.Type{unit_model.TypeActions}, nil, nil,
		)
		defer f()

		// create the forked repo
		forkedRepo, err := repo_service.ForkRepositoryAndUpdates(git.DefaultContext, user2, org3, repo_service.ForkRepoOptions{
			BaseRepo:    baseRepo,
			Name:        "forked-repo-pull-request-target",
			Description: "test pull-request-target event",
		})
		require.NoError(t, err)
		assert.NotEmpty(t, forkedRepo)

		// add workflow file to the base repo
		addWorkflowToBaseResp, err := files_service.ChangeRepoFiles(git.DefaultContext, baseRepo, user2, &files_service.ChangeRepoFilesOptions{
			Files: []*files_service.ChangeRepoFile{
				{
					Operation:     "create",
					TreePath:      ".gitea/workflows/pr.yml",
					ContentReader: strings.NewReader("name: test\non:\n  pull_request_target:\n    paths:\n      - 'file_*.txt'\njobs:\n  test:\n    runs-on: ubuntu-latest\n    steps:\n      - run: echo helloworld\n"),
				},
			},
			Message:   "add workflow",
			OldBranch: "main",
			NewBranch: "main",
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
		assert.NotEmpty(t, addWorkflowToBaseResp)

		// add a new file to the forked repo
		addFileToForkedResp, err := files_service.ChangeRepoFiles(git.DefaultContext, forkedRepo, org3, &files_service.ChangeRepoFilesOptions{
			Files: []*files_service.ChangeRepoFile{
				{
					Operation:     "create",
					TreePath:      "file_1.txt",
					ContentReader: strings.NewReader("file1"),
				},
			},
			Message:   "add file1",
			OldBranch: "main",
			NewBranch: "fork-branch-1",
			Author: &files_service.IdentityOptions{
				Name:  org3.Name,
				Email: org3.Email,
			},
			Committer: &files_service.IdentityOptions{
				Name:  org3.Name,
				Email: org3.Email,
			},
			Dates: &files_service.CommitDateOptions{
				Author:    time.Now(),
				Committer: time.Now(),
			},
		})
		require.NoError(t, err)
		assert.NotEmpty(t, addFileToForkedResp)

		// create Pull
		pullIssue := &issues_model.Issue{
			RepoID:   baseRepo.ID,
			Title:    "Test pull-request-target-event",
			PosterID: org3.ID,
			Poster:   org3,
			IsPull:   true,
		}
		pullRequest := &issues_model.PullRequest{
			HeadRepoID: forkedRepo.ID,
			BaseRepoID: baseRepo.ID,
			HeadBranch: "fork-branch-1",
			BaseBranch: "main",
			HeadRepo:   forkedRepo,
			BaseRepo:   baseRepo,
			Type:       issues_model.PullRequestGitea,
		}
		err = pull_service.NewPullRequest(git.DefaultContext, baseRepo, pullIssue, nil, nil, pullRequest, nil)
		require.NoError(t, err)
		// if a PR "synchronized" event races the "opened" event by having the same SHA, it must be skipped. See https://codeberg.org/forgejo/forgejo/issues/2009.
		assert.True(t, actions_service.SkipPullRequestEvent(git.DefaultContext, webhook_module.HookEventPullRequestSync, baseRepo.ID, addFileToForkedResp.Commit.SHA))

		// load and compare ActionRun
		assert.Equal(t, 1, unittest.GetCount(t, &actions_model.ActionRun{RepoID: baseRepo.ID}))
		actionRun := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRun{RepoID: baseRepo.ID})
		assert.Equal(t, addFileToForkedResp.Commit.SHA, actionRun.CommitSHA)
		assert.Equal(t, actions_module.GithubEventPullRequestTarget, actionRun.TriggerEvent)

		// add another file whose name cannot match the specified path
		addFileToForkedResp, err = files_service.ChangeRepoFiles(git.DefaultContext, forkedRepo, org3, &files_service.ChangeRepoFilesOptions{
			Files: []*files_service.ChangeRepoFile{
				{
					Operation:     "create",
					TreePath:      "foo.txt",
					ContentReader: strings.NewReader("foo"),
				},
			},
			Message:   "add foo.txt",
			OldBranch: "main",
			NewBranch: "fork-branch-2",
			Author: &files_service.IdentityOptions{
				Name:  org3.Name,
				Email: org3.Email,
			},
			Committer: &files_service.IdentityOptions{
				Name:  org3.Name,
				Email: org3.Email,
			},
			Dates: &files_service.CommitDateOptions{
				Author:    time.Now(),
				Committer: time.Now(),
			},
		})
		require.NoError(t, err)
		assert.NotEmpty(t, addFileToForkedResp)

		// create Pull
		pullIssue = &issues_model.Issue{
			RepoID:   baseRepo.ID,
			Title:    "A mismatched path cannot trigger pull-request-target-event",
			PosterID: org3.ID,
			Poster:   org3,
			IsPull:   true,
		}
		pullRequest = &issues_model.PullRequest{
			HeadRepoID: forkedRepo.ID,
			BaseRepoID: baseRepo.ID,
			HeadBranch: "fork-branch-2",
			BaseBranch: "main",
			HeadRepo:   forkedRepo,
			BaseRepo:   baseRepo,
			Type:       issues_model.PullRequestGitea,
		}
		err = pull_service.NewPullRequest(git.DefaultContext, baseRepo, pullIssue, nil, nil, pullRequest, nil)
		require.NoError(t, err)

		// the new pull request cannot trigger actions, so there is still only 1 record
		assert.Equal(t, 1, unittest.GetCount(t, &actions_model.ActionRun{RepoID: baseRepo.ID}))
	})
}

func TestSkipCI(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		session := loginUser(t, "user2")
		user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

		// create the repo
		repo, _, f := tests.CreateDeclarativeRepo(t, user2, "skip-ci",
			[]unit_model.Type{unit_model.TypeActions}, nil,
			[]*files_service.ChangeRepoFile{
				{
					Operation:     "create",
					TreePath:      ".gitea/workflows/pr.yml",
					ContentReader: strings.NewReader("name: test\non:\n  push:\n    branches: [main]\n  pull_request:\njobs:\n  test:\n    runs-on: ubuntu-latest\n    steps:\n      - run: echo helloworld\n"),
				},
			},
		)
		defer f()

		// a run has been created
		assert.Equal(t, 1, unittest.GetCount(t, &actions_model.ActionRun{RepoID: repo.ID}))

		// add a file with a configured skip-ci string in commit message
		addFileResp, err := files_service.ChangeRepoFiles(git.DefaultContext, repo, user2, &files_service.ChangeRepoFilesOptions{
			Files: []*files_service.ChangeRepoFile{
				{
					Operation:     "create",
					TreePath:      "bar.txt",
					ContentReader: strings.NewReader("bar"),
				},
			},
			Message:   fmt.Sprintf("%s add bar", setting.Actions.SkipWorkflowStrings[0]),
			OldBranch: "main",
			NewBranch: "main",
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
		assert.NotEmpty(t, addFileResp)

		// the commit message contains a configured skip-ci string, so there is still only 1 record
		assert.Equal(t, 1, unittest.GetCount(t, &actions_model.ActionRun{RepoID: repo.ID}))

		// add file to new branch
		addFileToBranchResp, err := files_service.ChangeRepoFiles(git.DefaultContext, repo, user2, &files_service.ChangeRepoFilesOptions{
			Files: []*files_service.ChangeRepoFile{
				{
					Operation:     "create",
					TreePath:      "test-skip-ci",
					ContentReader: strings.NewReader("test-skip-ci"),
				},
			},
			Message:   "add test file",
			OldBranch: "main",
			NewBranch: "test-skip-ci",
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
		assert.NotEmpty(t, addFileToBranchResp)

		resp := testPullCreate(t, session, "user2", "skip-ci", true, "main", "test-skip-ci", "[skip ci] test-skip-ci")

		// check the redirected URL
		url := test.RedirectURL(resp)
		assert.Regexp(t, "^/user2/skip-ci/pulls/[0-9]*$", url)

		// the pr title contains a configured skip-ci string, so there is still only 1 record
		assert.Equal(t, 1, unittest.GetCount(t, &actions_model.ActionRun{RepoID: repo.ID}))
	})
}

func TestCreateDeleteRefEvent(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

		// create the repo
		repo, err := repo_service.CreateRepository(db.DefaultContext, user2, user2, repo_service.CreateRepoOptions{
			Name:          "create-delete-ref-event",
			Description:   "test create delete ref ci event",
			AutoInit:      true,
			Gitignores:    "Go",
			License:       "MIT",
			Readme:        "Default",
			DefaultBranch: "main",
			IsPrivate:     false,
		})
		require.NoError(t, err)
		assert.NotEmpty(t, repo)

		// enable actions
		err = repo_service.UpdateRepositoryUnits(db.DefaultContext, repo, []repo_model.RepoUnit{{
			RepoID: repo.ID,
			Type:   unit_model.TypeActions,
		}}, nil)
		require.NoError(t, err)

		// add workflow file to the repo
		addWorkflowToBaseResp, err := files_service.ChangeRepoFiles(git.DefaultContext, repo, user2, &files_service.ChangeRepoFilesOptions{
			Files: []*files_service.ChangeRepoFile{
				{
					Operation:     "create",
					TreePath:      ".gitea/workflows/createdelete.yml",
					ContentReader: strings.NewReader("name: test\non:\n  [create,delete]\njobs:\n  test:\n    runs-on: ubuntu-latest\n    steps:\n      - run: echo helloworld\n"),
				},
			},
			Message:   "add workflow",
			OldBranch: "main",
			NewBranch: "main",
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
		assert.NotEmpty(t, addWorkflowToBaseResp)

		// Get the commit ID of the default branch
		gitRepo, err := gitrepo.OpenRepository(git.DefaultContext, repo)
		require.NoError(t, err)
		defer gitRepo.Close()
		branch, err := git_model.GetBranch(db.DefaultContext, repo.ID, repo.DefaultBranch)
		require.NoError(t, err)

		// create a branch
		err = repo_service.CreateNewBranchFromCommit(db.DefaultContext, user2, repo, gitRepo, branch.CommitID, "test-create-branch")
		require.NoError(t, err)
		run := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRun{
			Title:      "add workflow",
			RepoID:     repo.ID,
			Event:      "create",
			Ref:        "refs/heads/test-create-branch",
			WorkflowID: "createdelete.yml",
			CommitSHA:  branch.CommitID,
		})
		assert.NotNil(t, run)

		// create a tag
		err = release_service.CreateNewTag(db.DefaultContext, user2, repo, branch.CommitID, "test-create-tag", "test create tag event")
		require.NoError(t, err)
		run = unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRun{
			Title:      "add workflow",
			RepoID:     repo.ID,
			Event:      "create",
			Ref:        "refs/tags/test-create-tag",
			WorkflowID: "createdelete.yml",
			CommitSHA:  branch.CommitID,
		})
		assert.NotNil(t, run)

		// delete the branch
		err = repo_service.DeleteBranch(db.DefaultContext, user2, repo, gitRepo, "test-create-branch")
		require.NoError(t, err)
		run = unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRun{
			Title:      "add workflow",
			RepoID:     repo.ID,
			Event:      "delete",
			Ref:        "refs/heads/main",
			WorkflowID: "createdelete.yml",
			CommitSHA:  branch.CommitID,
		})
		assert.NotNil(t, run)

		// delete the tag
		tag, err := repo_model.GetRelease(db.DefaultContext, repo.ID, "test-create-tag")
		require.NoError(t, err)
		err = release_service.DeleteReleaseByID(db.DefaultContext, repo, tag, user2, true)
		require.NoError(t, err)
		run = unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRun{
			Title:      "add workflow",
			RepoID:     repo.ID,
			Event:      "delete",
			Ref:        "refs/heads/main",
			WorkflowID: "createdelete.yml",
			CommitSHA:  branch.CommitID,
		})
		assert.NotNil(t, run)
	})
}

func TestWorkflowDispatchEvent(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

		// create the repo
		repo, sha, f := tests.CreateDeclarativeRepo(t, user2, "repo-workflow-dispatch",
			[]unit_model.Type{unit_model.TypeActions}, nil,
			[]*files_service.ChangeRepoFile{
				{
					Operation: "create",
					TreePath:  ".gitea/workflows/dispatch.yml",
					ContentReader: strings.NewReader(
						"name: test\n" +
							"on: [workflow_dispatch]\n" +
							"jobs:\n" +
							"  test:\n" +
							"    runs-on: ubuntu-latest\n" +
							"    steps:\n" +
							"      - run: echo helloworld\n",
					),
				},
			},
		)
		defer f()

		gitRepo, err := gitrepo.OpenRepository(db.DefaultContext, repo)
		require.NoError(t, err)
		defer gitRepo.Close()

		workflow, err := actions_service.GetWorkflowFromCommit(gitRepo, "main", "dispatch.yml")
		require.NoError(t, err)
		assert.Equal(t, "refs/heads/main", workflow.Ref)
		assert.Equal(t, sha, workflow.Commit.ID.String())

		inputGetter := func(key string) string {
			return ""
		}

		var r *actions_model.ActionRun
		var j []string
		r, j, err = workflow.Dispatch(db.DefaultContext, inputGetter, repo, user2)
		require.NoError(t, err)

		assert.Equal(t, 1, unittest.GetCount(t, &actions_model.ActionRun{RepoID: repo.ID}))

		assert.Equal(t, "test", r.Title)
		assert.Equal(t, "dispatch.yml", r.WorkflowID)
		assert.Equal(t, sha, r.CommitSHA)
		assert.Equal(t, actions_module.GithubEventWorkflowDispatch, r.TriggerEvent)
		assert.Len(t, j, 1)
		assert.Equal(t, "test", j[0])
	})
}
