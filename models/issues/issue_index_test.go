// Copyright 2024 The Forgejo Authors
// SPDX-License-Identifier: MIT

package issues_test

import (
	"testing"

	"forgejo.org/models/db"
	issues_model "forgejo.org/models/issues"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetMaxIssueIndexForRepo(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})

	maxPR, err := issues_model.GetMaxIssueIndexForRepo(db.DefaultContext, repo.ID)
	require.NoError(t, err)

	issue := testCreateIssue(t, repo.ID, repo.OwnerID, "title1", "content1", false)
	assert.Greater(t, issue.Index, maxPR)

	maxPR, err = issues_model.GetMaxIssueIndexForRepo(db.DefaultContext, repo.ID)
	require.NoError(t, err)

	pull := testCreateIssue(t, repo.ID, repo.OwnerID, "title2", "content2", true)
	assert.Greater(t, pull.Index, maxPR)

	maxPR, err = issues_model.GetMaxIssueIndexForRepo(db.DefaultContext, repo.ID)
	require.NoError(t, err)

	assert.Equal(t, maxPR, pull.Index)
}
