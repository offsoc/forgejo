// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issues_test

import (
	"testing"

	"forgejo.org/models/db"
	issues_model "forgejo.org/models/issues"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIssueNewIssueLabels(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 2})
	label1 := unittest.AssertExistsAndLoadBean(t, &issues_model.Label{ID: 1})
	label2 := unittest.AssertExistsAndLoadBean(t, &issues_model.Label{ID: 4})
	doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	label3 := &issues_model.Label{RepoID: 1, Name: "label3", Color: "#123"}
	require.NoError(t, issues_model.NewLabel(db.DefaultContext, label3))

	// label1 is already set, do nothing
	// label3 is new, add it
	require.NoError(t, issues_model.NewIssueLabels(db.DefaultContext, issue, []*issues_model.Label{label1, label3}, doer))

	assert.Len(t, issue.Labels, 3)
	// check that the pre-existing label1 is still present
	assert.Equal(t, label1.ID, issue.Labels[0].ID)
	// check that new label3 was added
	assert.Equal(t, label3.ID, issue.Labels[1].ID)
	// check that pre-existing label2 was not removed
	assert.Equal(t, label2.ID, issue.Labels[2].ID)
}

func TestIssueNewIssueLabel(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 3})
	doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	label := &issues_model.Label{RepoID: 1, Name: "label3", Color: "#123"}
	require.NoError(t, issues_model.NewLabel(db.DefaultContext, label))

	require.NoError(t, issues_model.NewIssueLabel(db.DefaultContext, issue, label, doer))

	assert.Len(t, issue.Labels, 1)
	assert.Equal(t, label.ID, issue.Labels[0].ID)
}

func TestIssueReplaceIssueLabels(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 2})
	label1 := unittest.AssertExistsAndLoadBean(t, &issues_model.Label{ID: 1})
	label2 := unittest.AssertExistsAndLoadBean(t, &issues_model.Label{ID: 4})
	doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	label3 := &issues_model.Label{RepoID: 1, Name: "label3", Color: "#123"}
	require.NoError(t, issues_model.NewLabel(db.DefaultContext, label3))

	issue.LoadLabels(db.DefaultContext)
	assert.Len(t, issue.Labels, 2)
	assert.Equal(t, label1.ID, issue.Labels[0].ID)
	assert.Equal(t, label2.ID, issue.Labels[1].ID)

	// label1 is already set, do nothing
	// label3 is new, add it
	// label2 is not in the list but already set, remove it
	require.NoError(t, issues_model.ReplaceIssueLabels(db.DefaultContext, issue, []*issues_model.Label{label1, label3}, doer))

	assert.Len(t, issue.Labels, 2)
	assert.Equal(t, label1.ID, issue.Labels[0].ID)
	assert.Equal(t, label3.ID, issue.Labels[1].ID)
}

func TestIssueDeleteIssueLabel(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 2})
	label1 := unittest.AssertExistsAndLoadBean(t, &issues_model.Label{ID: 1})
	label2 := unittest.AssertExistsAndLoadBean(t, &issues_model.Label{ID: 4})
	doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	issue.LoadLabels(db.DefaultContext)
	assert.Len(t, issue.Labels, 2)
	assert.Equal(t, label1.ID, issue.Labels[0].ID)
	assert.Equal(t, label2.ID, issue.Labels[1].ID)

	require.NoError(t, issues_model.DeleteIssueLabel(db.DefaultContext, issue, label2, doer))

	assert.Len(t, issue.Labels, 1)
	assert.Equal(t, label1.ID, issue.Labels[0].ID)
}

func TestIssueLoadLabels(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 2})
	label1 := unittest.AssertExistsAndLoadBean(t, &issues_model.Label{ID: 1})
	label2 := unittest.AssertExistsAndLoadBean(t, &issues_model.Label{ID: 4})

	assert.Empty(t, issue.Labels)
	issue.LoadLabels(db.DefaultContext)
	assert.Len(t, issue.Labels, 2)
	assert.Equal(t, label1.ID, issue.Labels[0].ID)
	assert.Equal(t, label2.ID, issue.Labels[1].ID)

	unittest.AssertSuccessfulDelete(t, &issues_model.IssueLabel{IssueID: issue.ID, LabelID: label2.ID})

	// the database change is not noticed because the labels are cached
	issue.LoadLabels(db.DefaultContext)
	assert.Len(t, issue.Labels, 2)

	issue.ReloadLabels(db.DefaultContext)
	assert.Len(t, issue.Labels, 1)
	assert.Equal(t, label1.ID, issue.Labels[0].ID)
}

func TestNewIssueLabelsScope(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 18})
	label1 := unittest.AssertExistsAndLoadBean(t, &issues_model.Label{ID: 7})
	label2 := unittest.AssertExistsAndLoadBean(t, &issues_model.Label{ID: 8})
	doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	require.NoError(t, issues_model.NewIssueLabels(db.DefaultContext, issue, []*issues_model.Label{label1, label2}, doer))

	assert.Len(t, issue.Labels, 1)
	assert.Equal(t, label2.ID, issue.Labels[0].ID)
}
