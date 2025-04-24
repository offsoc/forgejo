// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"

	actions_model "forgejo.org/models/actions"
	"forgejo.org/models/db"
	unit_model "forgejo.org/models/unit"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/gitrepo"
	"forgejo.org/modules/setting"
	actions_service "forgejo.org/services/actions"
	notify_service "forgejo.org/services/notify"
	files_service "forgejo.org/services/repository/files"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockNotifier struct {
	notify_service.NullNotifier
	testIdx   int
	t         *testing.T
	runID     int64
	lastRunID int64
}

var _ notify_service.Notifier = &mockNotifier{}

func (m *mockNotifier) ActionRunNowDone(ctx context.Context, run *actions_model.ActionRun, priorStatus actions_model.Status, lastRun *actions_model.ActionRun) {
	switch m.testIdx {
	case 0:
		// we accept the first id as okay and just check that the following ones make sense
		m.runID = run.ID
		assert.Equal(m.t, actions_model.StatusSuccess, run.Status)
		assert.Equal(m.t, actions_model.StatusRunning, priorStatus)
		assert.Nil(m.t, lastRun)
	case 1:
		assert.Equal(m.t, m.runID, run.ID)
		assert.Equal(m.t, actions_model.StatusFailure, run.Status)
		assert.Equal(m.t, actions_model.StatusRunning, priorStatus)
		assert.Equal(m.t, m.lastRunID, lastRun.ID)
		assert.Equal(m.t, actions_model.StatusSuccess, lastRun.Status)
	case 2:
		assert.Equal(m.t, m.runID, run.ID)
		assert.Equal(m.t, actions_model.StatusCancelled, run.Status)
		assert.Equal(m.t, actions_model.StatusRunning, priorStatus)
		assert.Equal(m.t, m.lastRunID, lastRun.ID)
		assert.Equal(m.t, actions_model.StatusFailure, lastRun.Status)
	case 3:
		assert.Equal(m.t, m.runID, run.ID)
		assert.Equal(m.t, actions_model.StatusSuccess, run.Status)
		assert.Equal(m.t, actions_model.StatusRunning, priorStatus)
		assert.Equal(m.t, m.lastRunID, lastRun.ID)
		assert.Equal(m.t, actions_model.StatusCancelled, lastRun.Status)
	case 4:
		assert.Equal(m.t, m.runID, run.ID)
		assert.Equal(m.t, actions_model.StatusSuccess, run.Status)
		assert.Equal(m.t, actions_model.StatusRunning, priorStatus)
		assert.Equal(m.t, m.lastRunID, lastRun.ID)
		assert.Equal(m.t, actions_model.StatusSuccess, lastRun.Status)
	default:
		assert.Fail(m.t, "too many notifications")
	}
	m.lastRunID = m.runID
	m.runID++
	m.testIdx++
}

// ensure all tests have been run
func (m *mockNotifier) complete() {
	assert.Equal(m.t, 5, m.testIdx)
}

func TestActionNowDoneNotification(t *testing.T) {
	if !setting.Database.Type.IsSQLite3() {
		t.Skip()
	}

	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		notifier := mockNotifier{t: t, testIdx: 0, lastRunID: -1, runID: -1}
		notify_service.RegisterNotifier(&notifier)

		user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

		// create the repo
		repo, sha, f := tests.CreateDeclarativeRepo(t, user2, "repo-workflow-dispatch",
			[]unit_model.Type{unit_model.TypeActions}, nil,
			[]*files_service.ChangeRepoFile{
				{
					Operation: "create",
					TreePath:  ".forgejo/workflows/dispatch.yml",
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

		runner := newMockRunner()
		runner.registerAsRepoRunner(t, user2.Name, repo.Name, "mock-runner", []string{"ubuntu-latest"})

		// 0: first successful run
		_, _, err = workflow.Dispatch(db.DefaultContext, inputGetter, repo, user2)
		require.NoError(t, err)
		task := runner.fetchTask(t)
		runner.succeedAtTask(t, task)

		// we can't differentiate different runs without a delay
		time.Sleep(time.Millisecond * 2000)

		// 1: failed run
		_, _, err = workflow.Dispatch(db.DefaultContext, inputGetter, repo, user2)
		require.NoError(t, err)
		task = runner.fetchTask(t)
		runner.failAtTask(t, task)

		// we can't differentiate different runs without a delay
		time.Sleep(time.Millisecond * 2000)

		// 2: canceled run
		_, _, err = workflow.Dispatch(db.DefaultContext, inputGetter, repo, user2)
		require.NoError(t, err)
		task = runner.fetchTask(t)
		require.NoError(t, actions_service.StopTask(db.DefaultContext, task.Id, actions_model.StatusCancelled))

		// we can't differentiate different runs without a delay
		time.Sleep(time.Millisecond * 2000)

		// 3: successful run after failure
		_, _, err = workflow.Dispatch(db.DefaultContext, inputGetter, repo, user2)
		require.NoError(t, err)
		task = runner.fetchTask(t)
		runner.succeedAtTask(t, task)

		// we can't differentiate different runs without a delay
		time.Sleep(time.Millisecond * 2000)

		// 4: successful run after success
		_, _, err = workflow.Dispatch(db.DefaultContext, inputGetter, repo, user2)
		require.NoError(t, err)
		task = runner.fetchTask(t)
		runner.succeedAtTask(t, task)

		notifier.complete()
	})
}
