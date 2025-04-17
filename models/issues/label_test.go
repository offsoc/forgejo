// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issues_test

import (
	"testing"

	"forgejo.org/models/db"
	issues_model "forgejo.org/models/issues"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/timeutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLabel_CalOpenIssues(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	label := unittest.AssertExistsAndLoadBean(t, &issues_model.Label{ID: 1})
	label.CalOpenIssues()
	assert.Equal(t, 2, label.NumOpenIssues)
}

func TestLabel_LoadSelectedLabelsAfterClick(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	// Loading the label id:8 (scope/label2) which have a scope and an
	// exclusivity with id:7 (scope/label1)
	label := unittest.AssertExistsAndLoadBean(t, &issues_model.Label{ID: 8})

	// First test : with negative and scope
	label.LoadSelectedLabelsAfterClick([]int64{1, -8}, []string{"", "scope"})
	assert.Equal(t, "1", label.QueryString)
	assert.True(t, label.IsSelected)

	// Second test : with duplicates
	label.LoadSelectedLabelsAfterClick([]int64{1, 7, 1, 7, 7}, []string{"", "scope", "", "scope", "scope"})
	assert.Equal(t, "1,8", label.QueryString)
	assert.False(t, label.IsSelected)

	// Third test : empty set
	label.LoadSelectedLabelsAfterClick([]int64{}, []string{})
	assert.False(t, label.IsSelected)
	assert.Equal(t, "8", label.QueryString)
}

func TestLabel_ExclusiveScope(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	label := unittest.AssertExistsAndLoadBean(t, &issues_model.Label{ID: 7})
	assert.Equal(t, "scope", label.ExclusiveScope())

	label = unittest.AssertExistsAndLoadBean(t, &issues_model.Label{ID: 9})
	assert.Equal(t, "scope/subscope", label.ExclusiveScope())
}

func TestNewLabels(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	labels := []*issues_model.Label{
		{RepoID: 2, Name: "labelName2", Color: "#123456"},
		{RepoID: 3, Name: "labelName3", Color: "#123"},
		{RepoID: 4, Name: "labelName4", Color: "ABCDEF"},
		{RepoID: 5, Name: "labelName5", Color: "DEF"},
	}
	require.Error(t, issues_model.NewLabel(db.DefaultContext, &issues_model.Label{RepoID: 3, Name: "invalid Color", Color: ""}))
	require.Error(t, issues_model.NewLabel(db.DefaultContext, &issues_model.Label{RepoID: 3, Name: "invalid Color", Color: "#45G"}))
	require.Error(t, issues_model.NewLabel(db.DefaultContext, &issues_model.Label{RepoID: 3, Name: "invalid Color", Color: "#12345G"}))
	require.Error(t, issues_model.NewLabel(db.DefaultContext, &issues_model.Label{RepoID: 3, Name: "invalid Color", Color: "45G"}))
	require.Error(t, issues_model.NewLabel(db.DefaultContext, &issues_model.Label{RepoID: 3, Name: "invalid Color", Color: "12345G"}))
	for _, label := range labels {
		unittest.AssertNotExistsBean(t, label)
	}
	require.NoError(t, issues_model.NewLabels(db.DefaultContext, labels...))
	for _, label := range labels {
		unittest.AssertExistsAndLoadBean(t, label, unittest.Cond("id = ?", label.ID))
	}
	unittest.CheckConsistencyFor(t, &issues_model.Label{}, &repo_model.Repository{})
}

func TestGetLabelByID(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	label, err := issues_model.GetLabelByID(db.DefaultContext, 1)
	require.NoError(t, err)
	assert.EqualValues(t, 1, label.ID)

	_, err = issues_model.GetLabelByID(db.DefaultContext, unittest.NonexistentID)
	assert.True(t, issues_model.IsErrLabelNotExist(err))
}

func TestGetLabelInRepoByName(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	label, err := issues_model.GetLabelInRepoByName(db.DefaultContext, 1, "label1")
	require.NoError(t, err)
	assert.EqualValues(t, 1, label.ID)
	assert.Equal(t, "label1", label.Name)

	_, err = issues_model.GetLabelInRepoByName(db.DefaultContext, 1, "")
	assert.True(t, issues_model.IsErrRepoLabelNotExist(err))

	_, err = issues_model.GetLabelInRepoByName(db.DefaultContext, unittest.NonexistentID, "nonexistent")
	assert.True(t, issues_model.IsErrRepoLabelNotExist(err))
}

func TestGetLabelInRepoByNames(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	labelIDs, err := issues_model.GetLabelIDsInRepoByNames(db.DefaultContext, 1, []string{"label1", "label2"})
	require.NoError(t, err)

	assert.Len(t, labelIDs, 2)

	assert.Equal(t, int64(1), labelIDs[0])
	assert.Equal(t, int64(2), labelIDs[1])
}

func TestGetLabelInRepoByNamesDiscardsNonExistentLabels(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	// label3 doesn't exists.. See labels.yml
	labelIDs, err := issues_model.GetLabelIDsInRepoByNames(db.DefaultContext, 1, []string{"label1", "label2", "label3"})
	require.NoError(t, err)

	assert.Len(t, labelIDs, 2)

	assert.Equal(t, int64(1), labelIDs[0])
	assert.Equal(t, int64(2), labelIDs[1])
	require.NoError(t, err)
}

func TestGetLabelInRepoByID(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	label, err := issues_model.GetLabelInRepoByID(db.DefaultContext, 1, 1)
	require.NoError(t, err)
	assert.EqualValues(t, 1, label.ID)

	_, err = issues_model.GetLabelInRepoByID(db.DefaultContext, 1, -1)
	assert.True(t, issues_model.IsErrRepoLabelNotExist(err))

	_, err = issues_model.GetLabelInRepoByID(db.DefaultContext, unittest.NonexistentID, unittest.NonexistentID)
	assert.True(t, issues_model.IsErrRepoLabelNotExist(err))
}

func TestGetLabelsInRepoByIDs(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	labels, err := issues_model.GetLabelsInRepoByIDs(db.DefaultContext, 1, []int64{1, 2, unittest.NonexistentID})
	require.NoError(t, err)
	if assert.Len(t, labels, 2) {
		assert.EqualValues(t, 1, labels[0].ID)
		assert.EqualValues(t, 2, labels[1].ID)
	}
}

func TestGetLabelsByRepoID(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	testSuccess := func(repoID int64, sortType string, expectedIssueIDs []int64) {
		labels, err := issues_model.GetLabelsByRepoID(db.DefaultContext, repoID, sortType, db.ListOptions{})
		require.NoError(t, err)
		assert.Len(t, labels, len(expectedIssueIDs))
		for i, label := range labels {
			assert.Equal(t, expectedIssueIDs[i], label.ID)
		}
	}
	testSuccess(1, "leastissues", []int64{2, 1})
	testSuccess(1, "mostissues", []int64{1, 2})
	testSuccess(1, "reversealphabetically", []int64{2, 1})
	testSuccess(1, "default", []int64{1, 2})
}

// Org versions

func TestGetLabelInOrgByName(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	label, err := issues_model.GetLabelInOrgByName(db.DefaultContext, 3, "orglabel3")
	require.NoError(t, err)
	assert.EqualValues(t, 3, label.ID)
	assert.Equal(t, "orglabel3", label.Name)

	_, err = issues_model.GetLabelInOrgByName(db.DefaultContext, 3, "")
	assert.True(t, issues_model.IsErrOrgLabelNotExist(err))

	_, err = issues_model.GetLabelInOrgByName(db.DefaultContext, 0, "orglabel3")
	assert.True(t, issues_model.IsErrOrgLabelNotExist(err))

	_, err = issues_model.GetLabelInOrgByName(db.DefaultContext, -1, "orglabel3")
	assert.True(t, issues_model.IsErrOrgLabelNotExist(err))

	_, err = issues_model.GetLabelInOrgByName(db.DefaultContext, unittest.NonexistentID, "nonexistent")
	assert.True(t, issues_model.IsErrOrgLabelNotExist(err))
}

func TestGetLabelInOrgByID(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	label, err := issues_model.GetLabelInOrgByID(db.DefaultContext, 3, 3)
	require.NoError(t, err)
	assert.EqualValues(t, 3, label.ID)

	_, err = issues_model.GetLabelInOrgByID(db.DefaultContext, 3, -1)
	assert.True(t, issues_model.IsErrOrgLabelNotExist(err))

	_, err = issues_model.GetLabelInOrgByID(db.DefaultContext, 0, 3)
	assert.True(t, issues_model.IsErrOrgLabelNotExist(err))

	_, err = issues_model.GetLabelInOrgByID(db.DefaultContext, -1, 3)
	assert.True(t, issues_model.IsErrOrgLabelNotExist(err))

	_, err = issues_model.GetLabelInOrgByID(db.DefaultContext, unittest.NonexistentID, unittest.NonexistentID)
	assert.True(t, issues_model.IsErrOrgLabelNotExist(err))
}

func TestGetLabelsInOrgByIDs(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	labels, err := issues_model.GetLabelsInOrgByIDs(db.DefaultContext, 3, []int64{3, 4, unittest.NonexistentID})
	require.NoError(t, err)
	if assert.Len(t, labels, 2) {
		assert.EqualValues(t, 3, labels[0].ID)
		assert.EqualValues(t, 4, labels[1].ID)
	}
}

func TestGetLabelsByOrgID(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	testSuccess := func(orgID int64, sortType string, expectedIssueIDs []int64) {
		labels, err := issues_model.GetLabelsByOrgID(db.DefaultContext, orgID, sortType, db.ListOptions{})
		require.NoError(t, err)
		assert.Len(t, labels, len(expectedIssueIDs))
		for i, label := range labels {
			assert.Equal(t, expectedIssueIDs[i], label.ID)
		}
	}
	testSuccess(3, "leastissues", []int64{3, 4})
	testSuccess(3, "mostissues", []int64{4, 3})
	testSuccess(3, "reversealphabetically", []int64{4, 3})
	testSuccess(3, "default", []int64{3, 4})

	_, err := issues_model.GetLabelsByOrgID(db.DefaultContext, 0, "leastissues", db.ListOptions{})
	assert.True(t, issues_model.IsErrOrgLabelNotExist(err))

	_, err = issues_model.GetLabelsByOrgID(db.DefaultContext, -1, "leastissues", db.ListOptions{})
	assert.True(t, issues_model.IsErrOrgLabelNotExist(err))
}

//

func TestGetLabelsByIssueID(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	labels, err := issues_model.GetLabelsByIssueID(db.DefaultContext, 1)
	require.NoError(t, err)
	if assert.Len(t, labels, 1) {
		assert.EqualValues(t, 1, labels[0].ID)
	}

	labels, err = issues_model.GetLabelsByIssueID(db.DefaultContext, unittest.NonexistentID)
	require.NoError(t, err)
	assert.Empty(t, labels)
}

func TestUpdateLabel(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	label := unittest.AssertExistsAndLoadBean(t, &issues_model.Label{ID: 1})
	// make sure update won't overwrite it
	update := &issues_model.Label{
		ID:           label.ID,
		Color:        "#ffff00",
		Name:         "newLabelName",
		Description:  label.Description,
		Exclusive:    false,
		ArchivedUnix: timeutil.TimeStamp(0),
	}
	label.Color = update.Color
	label.Name = update.Name
	require.NoError(t, issues_model.UpdateLabel(db.DefaultContext, update))
	newLabel := unittest.AssertExistsAndLoadBean(t, &issues_model.Label{ID: 1})
	assert.Equal(t, label.ID, newLabel.ID)
	assert.Equal(t, label.Color, newLabel.Color)
	assert.Equal(t, label.Name, newLabel.Name)
	assert.Equal(t, label.Description, newLabel.Description)
	assert.EqualValues(t, 0, newLabel.ArchivedUnix)
	unittest.CheckConsistencyFor(t, &issues_model.Label{}, &repo_model.Repository{})
}

func TestDeleteLabel(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	label := unittest.AssertExistsAndLoadBean(t, &issues_model.Label{ID: 1})
	require.NoError(t, issues_model.DeleteLabel(db.DefaultContext, label.RepoID, label.ID))
	unittest.AssertNotExistsBean(t, &issues_model.Label{ID: label.ID, RepoID: label.RepoID})

	require.NoError(t, issues_model.DeleteLabel(db.DefaultContext, label.RepoID, label.ID))
	unittest.AssertNotExistsBean(t, &issues_model.Label{ID: label.ID})

	require.NoError(t, issues_model.DeleteLabel(db.DefaultContext, unittest.NonexistentID, unittest.NonexistentID))
	unittest.CheckConsistencyFor(t, &issues_model.Label{}, &repo_model.Repository{})
}

func TestHasIssueLabel(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	assert.True(t, issues_model.HasIssueLabel(db.DefaultContext, 1, 1))
	assert.False(t, issues_model.HasIssueLabel(db.DefaultContext, 1, 2))
	assert.False(t, issues_model.HasIssueLabel(db.DefaultContext, unittest.NonexistentID, unittest.NonexistentID))
}

func TestNewIssueLabel(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	label := unittest.AssertExistsAndLoadBean(t, &issues_model.Label{ID: 2})
	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 1})
	doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	// add new IssueLabel
	prevNumIssues := label.NumIssues
	require.NoError(t, issues_model.NewIssueLabel(db.DefaultContext, issue, label, doer))
	unittest.AssertExistsAndLoadBean(t, &issues_model.IssueLabel{IssueID: issue.ID, LabelID: label.ID})
	unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{
		Type:     issues_model.CommentTypeLabel,
		PosterID: doer.ID,
		IssueID:  issue.ID,
		LabelID:  label.ID,
		Content:  "1",
	})
	label = unittest.AssertExistsAndLoadBean(t, &issues_model.Label{ID: 2})
	assert.Equal(t, prevNumIssues+1, label.NumIssues)

	// re-add existing IssueLabel
	require.NoError(t, issues_model.NewIssueLabel(db.DefaultContext, issue, label, doer))
	unittest.CheckConsistencyFor(t, &issues_model.Issue{}, &issues_model.Label{})
}

func TestNewIssueExclusiveLabel(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 18})
	doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	otherLabel := unittest.AssertExistsAndLoadBean(t, &issues_model.Label{ID: 6})
	exclusiveLabelA := unittest.AssertExistsAndLoadBean(t, &issues_model.Label{ID: 7})
	exclusiveLabelB := unittest.AssertExistsAndLoadBean(t, &issues_model.Label{ID: 8})

	// coexisting regular and exclusive label
	require.NoError(t, issues_model.NewIssueLabel(db.DefaultContext, issue, otherLabel, doer))
	require.NoError(t, issues_model.NewIssueLabel(db.DefaultContext, issue, exclusiveLabelA, doer))
	unittest.AssertExistsAndLoadBean(t, &issues_model.IssueLabel{IssueID: issue.ID, LabelID: otherLabel.ID})
	unittest.AssertExistsAndLoadBean(t, &issues_model.IssueLabel{IssueID: issue.ID, LabelID: exclusiveLabelA.ID})

	// exclusive label replaces existing one
	require.NoError(t, issues_model.NewIssueLabel(db.DefaultContext, issue, exclusiveLabelB, doer))
	unittest.AssertExistsAndLoadBean(t, &issues_model.IssueLabel{IssueID: issue.ID, LabelID: otherLabel.ID})
	unittest.AssertExistsAndLoadBean(t, &issues_model.IssueLabel{IssueID: issue.ID, LabelID: exclusiveLabelB.ID})
	unittest.AssertNotExistsBean(t, &issues_model.IssueLabel{IssueID: issue.ID, LabelID: exclusiveLabelA.ID})

	// exclusive label replaces existing one again
	require.NoError(t, issues_model.NewIssueLabel(db.DefaultContext, issue, exclusiveLabelA, doer))
	unittest.AssertExistsAndLoadBean(t, &issues_model.IssueLabel{IssueID: issue.ID, LabelID: otherLabel.ID})
	unittest.AssertExistsAndLoadBean(t, &issues_model.IssueLabel{IssueID: issue.ID, LabelID: exclusiveLabelA.ID})
	unittest.AssertNotExistsBean(t, &issues_model.IssueLabel{IssueID: issue.ID, LabelID: exclusiveLabelB.ID})
}

func TestNewIssueLabels(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	label1 := unittest.AssertExistsAndLoadBean(t, &issues_model.Label{ID: 1})
	label2 := unittest.AssertExistsAndLoadBean(t, &issues_model.Label{ID: 2})
	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 5})
	doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	require.NoError(t, issues_model.NewIssueLabels(db.DefaultContext, issue, []*issues_model.Label{label1, label2}, doer))
	unittest.AssertExistsAndLoadBean(t, &issues_model.IssueLabel{IssueID: issue.ID, LabelID: label1.ID})
	unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{
		Type:     issues_model.CommentTypeLabel,
		PosterID: doer.ID,
		IssueID:  issue.ID,
		LabelID:  label1.ID,
		Content:  "1",
	})
	unittest.AssertExistsAndLoadBean(t, &issues_model.IssueLabel{IssueID: issue.ID, LabelID: label1.ID})
	label1 = unittest.AssertExistsAndLoadBean(t, &issues_model.Label{ID: 1})
	assert.Equal(t, 3, label1.NumIssues)
	assert.Equal(t, 1, label1.NumClosedIssues)
	label2 = unittest.AssertExistsAndLoadBean(t, &issues_model.Label{ID: 2})
	assert.Equal(t, 1, label2.NumIssues)
	assert.Equal(t, 1, label2.NumClosedIssues)

	// corner case: test empty slice
	require.NoError(t, issues_model.NewIssueLabels(db.DefaultContext, issue, []*issues_model.Label{}, doer))

	unittest.CheckConsistencyFor(t, &issues_model.Issue{}, &issues_model.Label{})
}

func TestDeleteIssueLabel(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	testSuccess := func(labelID, issueID, doerID int64) {
		label := unittest.AssertExistsAndLoadBean(t, &issues_model.Label{ID: labelID})
		issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: issueID})
		doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: doerID})

		expectedNumIssues := label.NumIssues
		expectedNumClosedIssues := label.NumClosedIssues
		if unittest.BeanExists(t, &issues_model.IssueLabel{IssueID: issueID, LabelID: labelID}) {
			expectedNumIssues--
			if issue.IsClosed {
				expectedNumClosedIssues--
			}
		}

		ctx, committer, err := db.TxContext(db.DefaultContext)
		defer committer.Close()
		require.NoError(t, err)
		require.NoError(t, issues_model.DeleteIssueLabel(ctx, issue, label, doer))
		require.NoError(t, committer.Commit())

		unittest.AssertNotExistsBean(t, &issues_model.IssueLabel{IssueID: issueID, LabelID: labelID})
		unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{
			Type:     issues_model.CommentTypeLabel,
			PosterID: doerID,
			IssueID:  issueID,
			LabelID:  labelID,
		}, `content=""`)
		label = unittest.AssertExistsAndLoadBean(t, &issues_model.Label{ID: labelID})
		assert.Equal(t, expectedNumIssues, label.NumIssues)
		assert.Equal(t, expectedNumClosedIssues, label.NumClosedIssues)
	}
	testSuccess(1, 1, 2)
	testSuccess(2, 5, 2)
	testSuccess(1, 1, 2) // delete non-existent IssueLabel

	unittest.CheckConsistencyFor(t, &issues_model.Issue{}, &issues_model.Label{})
}
