// Copyright 2024 The Forgejo Authors
// SPDX-License-Identifier: MIT

package issues

import (
	"testing"

	"forgejo.org/models/db"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommentListLoadUser(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	issue := unittest.AssertExistsAndLoadBean(t, &Issue{})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: issue.RepoID})
	doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})

	for _, testCase := range []struct {
		poster   int64
		assignee int64
		user     *user_model.User
	}{
		{
			poster:   user_model.ActionsUserID,
			assignee: user_model.ActionsUserID,
			user:     user_model.NewActionsUser(),
		},
		{
			poster:   user_model.GhostUserID,
			assignee: user_model.GhostUserID,
			user:     user_model.NewGhostUser(),
		},
		{
			poster:   doer.ID,
			assignee: doer.ID,
			user:     doer,
		},
		{
			poster:   0,
			assignee: 0,
			user:     user_model.NewGhostUser(),
		},
		{
			poster:   -200,
			assignee: -200,
			user:     user_model.NewGhostUser(),
		},
		{
			poster:   200,
			assignee: 200,
			user:     user_model.NewGhostUser(),
		},
	} {
		t.Run(testCase.user.Name, func(t *testing.T) {
			comment, err := CreateComment(db.DefaultContext, &CreateCommentOptions{
				Type:    CommentTypeComment,
				Doer:    testCase.user,
				Repo:    repo,
				Issue:   issue,
				Content: "Hello",
			})
			assert.NoError(t, err)

			list := CommentList{comment}

			comment.PosterID = testCase.poster
			comment.Poster = nil
			assert.NoError(t, list.LoadPosters(db.DefaultContext))
			require.NotNil(t, comment.Poster)
			assert.Equal(t, testCase.user.ID, comment.Poster.ID)

			comment.AssigneeID = testCase.assignee
			comment.Assignee = nil
			require.NoError(t, list.loadAssignees(db.DefaultContext))
			require.NotNil(t, comment.Assignee)
			assert.Equal(t, testCase.user.ID, comment.Assignee.ID)
		})
	}
}
