// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package actions

import (
	"testing"

	actions_model "forgejo.org/models/actions"
	"forgejo.org/models/db"
	unittest "forgejo.org/models/unittest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_loadIsRefDeleted(t *testing.T) {
	unittest.PrepareTestEnv(t)

	runs, total, err := db.FindAndCount[actions_model.ActionRun](db.DefaultContext,
		actions_model.FindRunOptions{RepoID: 4, Ref: "refs/heads/test"})
	require.NoError(t, err)
	assert.Len(t, runs, 1)
	assert.EqualValues(t, 1, total)
	for _, run := range runs {
		assert.False(t, run.IsRefDeleted)
	}

	require.NoError(t, loadIsRefDeleted(db.DefaultContext, 4, runs))
	for _, run := range runs {
		assert.True(t, run.IsRefDeleted)
	}
}
