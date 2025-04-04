// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issues_test

import (
	"testing"
	"time"

	"forgejo.org/models/db"
	issues_model "forgejo.org/models/issues"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/structs"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateComment(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: issue.RepoID})
	doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})

	now := time.Now().Unix()
	comment, err := issues_model.CreateComment(db.DefaultContext, &issues_model.CreateCommentOptions{
		Type:    issues_model.CommentTypeComment,
		Doer:    doer,
		Repo:    repo,
		Issue:   issue,
		Content: "Hello",
	})
	require.NoError(t, err)
	then := time.Now().Unix()

	assert.Equal(t, issues_model.CommentTypeComment, comment.Type)
	assert.Equal(t, "Hello", comment.Content)
	assert.Equal(t, issue.ID, comment.IssueID)
	assert.Equal(t, doer.ID, comment.PosterID)
	unittest.AssertInt64InRange(t, now, then, int64(comment.CreatedUnix))
	unittest.AssertExistsAndLoadBean(t, comment) // assert actually added to DB

	updatedIssue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: issue.ID})
	unittest.AssertInt64InRange(t, now, then, int64(updatedIssue.UpdatedUnix))
}

func TestFetchCodeConversations(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 2})
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
	res, err := issues_model.FetchCodeConversations(db.DefaultContext, issue, user, false)
	require.NoError(t, err)
	assert.Contains(t, res, "README.md")
	assert.Contains(t, res["README.md"], int64(4))
	assert.Len(t, res["README.md"][4], 1)
	assert.Equal(t, int64(4), res["README.md"][4][0][0].ID)

	user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	res, err = issues_model.FetchCodeConversations(db.DefaultContext, issue, user2, false)
	require.NoError(t, err)
	assert.Len(t, res, 1)
}

func TestAsCommentType(t *testing.T) {
	assert.Equal(t, issues_model.CommentTypeComment, issues_model.CommentType(0))
	assert.Equal(t, issues_model.CommentTypeUndefined, issues_model.AsCommentType(""))
	assert.Equal(t, issues_model.CommentTypeUndefined, issues_model.AsCommentType("nonsense"))
	assert.Equal(t, issues_model.CommentTypeComment, issues_model.AsCommentType("comment"))
	assert.Equal(t, issues_model.CommentTypePRUnScheduledToAutoMerge, issues_model.AsCommentType("pull_cancel_scheduled_merge"))
}

func TestMigrate_InsertIssueComments(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 1})
	_ = issue.LoadRepo(db.DefaultContext)
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: issue.Repo.OwnerID})
	reaction := &issues_model.Reaction{
		Type:   "heart",
		UserID: owner.ID,
	}

	comment := &issues_model.Comment{
		PosterID:  owner.ID,
		Poster:    owner,
		IssueID:   issue.ID,
		Issue:     issue,
		Reactions: []*issues_model.Reaction{reaction},
	}

	err := issues_model.InsertIssueComments(db.DefaultContext, []*issues_model.Comment{comment})
	require.NoError(t, err)

	issueModified := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 1})
	assert.Equal(t, issue.NumComments+1, issueModified.NumComments)

	unittest.CheckConsistencyFor(t, &issues_model.Issue{})
}

func TestUpdateCommentsMigrationsByType(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 1})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: issue.RepoID})
	comment := unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{ID: 1, IssueID: issue.ID})

	// Set repository to migrated from Gitea.
	repo.OriginalServiceType = structs.GiteaService
	repo_model.UpdateRepositoryCols(db.DefaultContext, repo, "original_service_type")

	// Set comment to have an original author.
	comment.OriginalAuthor = "Example User"
	comment.OriginalAuthorID = 1
	comment.PosterID = 0
	_, err := db.GetEngine(db.DefaultContext).ID(comment.ID).Cols("original_author", "original_author_id", "poster_id").Update(comment)
	require.NoError(t, err)

	require.NoError(t, issues_model.UpdateCommentsMigrationsByType(db.DefaultContext, structs.GiteaService, "1", 513))

	comment = unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{ID: 1, IssueID: issue.ID})
	assert.Empty(t, comment.OriginalAuthor)
	assert.Empty(t, comment.OriginalAuthorID)
	assert.EqualValues(t, 513, comment.PosterID)
}

func Test_UpdateIssueNumComments(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	issue2 := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 2})

	require.NoError(t, issues_model.UpdateIssueNumComments(db.DefaultContext, issue2.ID))
	issue2 = unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 2})
	assert.Equal(t, 1, issue2.NumComments)
}
