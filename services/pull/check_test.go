// Copyright 2019 The Gitea Authors.
// All rights reserved.
// SPDX-License-Identifier: MIT

package pull

import (
	"strconv"
	"testing"
	"time"

	"forgejo.org/models/db"
	issues_model "forgejo.org/models/issues"
	"forgejo.org/models/unittest"
	"forgejo.org/modules/queue"
	"forgejo.org/modules/setting"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPullRequest_AddToTaskQueue(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	idChan := make(chan int64, 10)
	testHandler := func(items ...string) []string {
		for _, s := range items {
			id, _ := strconv.ParseInt(s, 10, 64)
			idChan <- id
		}
		return nil
	}

	cfg, err := setting.GetQueueSettings(setting.CfgProvider, "pr_patch_checker")
	require.NoError(t, err)
	prPatchCheckerQueue, err = queue.NewWorkerPoolQueueWithContext(t.Context(), "pr_patch_checker", cfg, testHandler, true)
	require.NoError(t, err)

	pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 2})
	AddToTaskQueue(db.DefaultContext, pr)

	assert.Eventually(t, func() bool {
		pr = unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 2})
		return pr.Status == issues_model.PullRequestStatusChecking
	}, 1*time.Second, 100*time.Millisecond)

	has, err := prPatchCheckerQueue.Has(strconv.FormatInt(pr.ID, 10))
	assert.True(t, has)
	require.NoError(t, err)

	go prPatchCheckerQueue.Run()

	select {
	case id := <-idChan:
		assert.Equal(t, pr.ID, id)
	case <-time.After(time.Second):
		assert.FailNow(t, "Timeout: nothing was added to pullRequestQueue")
	}

	has, err = prPatchCheckerQueue.Has(strconv.FormatInt(pr.ID, 10))
	assert.False(t, has)
	require.NoError(t, err)

	pr = unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 2})
	assert.Equal(t, issues_model.PullRequestStatusChecking, pr.Status)

	prPatchCheckerQueue.ShutdownWait(5 * time.Second)
	prPatchCheckerQueue = nil
}
