// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package convert

import (
	"fmt"
	"testing"
	"time"

	issues_model "forgejo.org/models/issues"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	"forgejo.org/modules/setting"
	api "forgejo.org/modules/structs"
	"forgejo.org/modules/timeutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLabel_ToLabel(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	label := unittest.AssertExistsAndLoadBean(t, &issues_model.Label{ID: 1})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: label.RepoID})
	assert.Equal(t, &api.Label{
		ID:    label.ID,
		Name:  label.Name,
		Color: "abcdef",
		URL:   fmt.Sprintf("%sapi/v1/repos/user2/repo1/labels/%d", setting.AppURL, label.ID),
	}, ToLabel(label, repo, nil))
}

func TestMilestone_APIFormat(t *testing.T) {
	milestone := &issues_model.Milestone{
		ID:              3,
		RepoID:          4,
		Name:            "milestoneName",
		Content:         "milestoneContent",
		IsClosed:        false,
		NumOpenIssues:   5,
		NumClosedIssues: 6,
		CreatedUnix:     timeutil.TimeStamp(time.Date(1999, time.January, 1, 0, 0, 0, 0, time.UTC).Unix()),
		UpdatedUnix:     timeutil.TimeStamp(time.Date(1999, time.March, 1, 0, 0, 0, 0, time.UTC).Unix()),
		DeadlineUnix:    timeutil.TimeStamp(time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC).Unix()),
	}
	assert.Equal(t, api.Milestone{
		ID:           milestone.ID,
		State:        api.StateOpen,
		Title:        milestone.Name,
		Description:  milestone.Content,
		OpenIssues:   milestone.NumOpenIssues,
		ClosedIssues: milestone.NumClosedIssues,
		Created:      milestone.CreatedUnix.AsTime(),
		Updated:      milestone.UpdatedUnix.AsTimePtr(),
		Deadline:     milestone.DeadlineUnix.AsTimePtr(),
	}, *ToAPIMilestone(milestone))
}
