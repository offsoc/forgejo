// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package convert

import (
	"testing"

	"forgejo.org/models/db"
	issues_model "forgejo.org/models/issues"
	"forgejo.org/models/perm"
	access_model "forgejo.org/models/perm/access"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/git"
	"forgejo.org/modules/structs"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPullRequest_APIFormat(t *testing.T) {
	// with HeadRepo
	require.NoError(t, unittest.PrepareTestDatabase())
	headRepo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 1})
	require.NoError(t, pr.LoadAttributes(db.DefaultContext))
	require.NoError(t, pr.LoadIssue(db.DefaultContext))
	apiPullRequest := ToAPIPullRequest(git.DefaultContext, pr, nil)
	assert.NotNil(t, apiPullRequest)
	assert.Equal(t, &structs.PRBranchInfo{
		Name:       "branch1",
		Ref:        "refs/pull/2/head",
		Sha:        "4a357436d925b5c974181ff12a994538ddc5a269",
		RepoID:     1,
		Repository: ToRepo(db.DefaultContext, headRepo, access_model.Permission{AccessMode: perm.AccessModeRead}),
	}, apiPullRequest.Head)

	// withOut HeadRepo
	pr = unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 1})
	require.NoError(t, pr.LoadIssue(db.DefaultContext))
	require.NoError(t, pr.LoadAttributes(db.DefaultContext))
	// simulate fork deletion
	pr.HeadRepo = nil
	pr.HeadRepoID = 100000
	apiPullRequest = ToAPIPullRequest(git.DefaultContext, pr, nil)
	assert.NotNil(t, apiPullRequest)
	assert.Nil(t, apiPullRequest.Head.Repository)
	assert.EqualValues(t, -1, apiPullRequest.Head.RepoID)
}

func TestPullReviewList(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	t.Run("Pending review", func(t *testing.T) {
		reviewer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
		review := unittest.AssertExistsAndLoadBean(t, &issues_model.Review{ID: 6, ReviewerID: reviewer.ID})
		rl := []*issues_model.Review{review}

		t.Run("Anonymous", func(t *testing.T) {
			prList, err := ToPullReviewList(db.DefaultContext, rl, nil)
			require.NoError(t, err)
			assert.Empty(t, prList)
		})
		t.Run("Reviewer", func(t *testing.T) {
			prList, err := ToPullReviewList(db.DefaultContext, rl, reviewer)
			require.NoError(t, err)
			assert.Len(t, prList, 1)
		})
		t.Run("Admin", func(t *testing.T) {
			adminUser := unittest.AssertExistsAndLoadBean(t, &user_model.User{IsAdmin: true}, unittest.Cond("id != ?", reviewer.ID))
			prList, err := ToPullReviewList(db.DefaultContext, rl, adminUser)
			require.NoError(t, err)
			assert.Len(t, prList, 1)
		})
	})
}
