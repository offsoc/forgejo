// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issue_test

import (
	"testing"

	"forgejo.org/models/db"
	issues_model "forgejo.org/models/issues"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	webhook_model "forgejo.org/models/webhook"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/test"
	issue_service "forgejo.org/services/issue"
	"forgejo.org/tests"

	_ "forgejo.org/services/webhook"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeleteComment(t *testing.T) {
	// Use the webhook notification to check if a notification is fired for an action.
	defer test.MockVariableValue(&setting.DisableWebhooks, false)()
	require.NoError(t, unittest.PrepareTestDatabase())

	t.Run("Normal comment", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		comment := unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{ID: 2})
		issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: comment.IssueID})
		unittest.AssertCount(t, &issues_model.Reaction{CommentID: comment.ID}, 2)

		require.NoError(t, webhook_model.CreateWebhook(db.DefaultContext, &webhook_model.Webhook{
			RepoID:   issue.RepoID,
			IsActive: true,
			Events:   `{"choose_events":true,"events":{"issue_comment": true}}`,
		}))
		hookTaskCount := unittest.GetCount(t, &webhook_model.HookTask{})

		require.NoError(t, issue_service.DeleteComment(db.DefaultContext, nil, comment))

		// The comment doesn't exist anymore.
		unittest.AssertNotExistsBean(t, &issues_model.Comment{ID: comment.ID})
		// Reactions don't exist anymore for this comment.
		unittest.AssertNotExistsBean(t, &issues_model.Reaction{CommentID: comment.ID})
		// Number of comments was decreased.
		assert.Equal(t, issue.NumComments-1, unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: comment.IssueID}).NumComments)
		// A notification was fired for the deletion of this comment.
		assert.Equal(t, hookTaskCount+1, unittest.GetCount(t, &webhook_model.HookTask{}))
	})

	t.Run("Comment of pending review", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		// We have to ensure that this comment's linked review is pending.
		comment := unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{ID: 4}, "review_id != 0")
		review := unittest.AssertExistsAndLoadBean(t, &issues_model.Review{ID: comment.ReviewID})
		assert.Equal(t, issues_model.ReviewTypePending, review.Type)
		issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: comment.IssueID})

		require.NoError(t, webhook_model.CreateWebhook(db.DefaultContext, &webhook_model.Webhook{
			RepoID:   issue.RepoID,
			IsActive: true,
			Events:   `{"choose_events":true,"events":{"issue_comment": true}}`,
		}))
		hookTaskCount := unittest.GetCount(t, &webhook_model.HookTask{})

		require.NoError(t, comment.LoadReview(t.Context()))
		require.NoError(t, issue_service.DeleteComment(db.DefaultContext, nil, comment))

		// The comment doesn't exist anymore.
		unittest.AssertNotExistsBean(t, &issues_model.Comment{ID: comment.ID})
		// Ensure that the number of comments wasn't decreased.
		assert.Equal(t, issue.NumComments, unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: comment.IssueID}).NumComments)
		// No notification was fired for the deletion of this comment.
		assert.Equal(t, hookTaskCount, unittest.GetCount(t, &webhook_model.HookTask{}))
		// The review doesn't exist anymore.
		unittest.AssertNotExistsBean(t, &issues_model.Review{ID: comment.ReviewID})
	})
}

func TestUpdateComment(t *testing.T) {
	// Use the webhook notification to check if a notification is fired for an action.
	defer test.MockVariableValue(&setting.DisableWebhooks, false)()
	require.NoError(t, unittest.PrepareTestDatabase())

	admin := unittest.AssertExistsAndLoadBean(t, &user_model.User{IsAdmin: true})
	t.Run("Normal comment", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		comment := unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{ID: 2})
		issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: comment.IssueID})
		unittest.AssertNotExistsBean(t, &issues_model.ContentHistory{CommentID: comment.ID})
		require.NoError(t, webhook_model.CreateWebhook(db.DefaultContext, &webhook_model.Webhook{
			RepoID:   issue.RepoID,
			IsActive: true,
			Events:   `{"choose_events":true,"events":{"issue_comment": true}}`,
		}))
		hookTaskCount := unittest.GetCount(t, &webhook_model.HookTask{})
		oldContent := comment.Content
		comment.Content = "Hello!"

		require.NoError(t, issue_service.UpdateComment(db.DefaultContext, comment, 1, admin, oldContent))

		newComment := unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{ID: 2})
		// Content was updated.
		assert.Equal(t, comment.Content, newComment.Content)
		// Content version was updated.
		assert.Equal(t, 2, newComment.ContentVersion)
		// A notification was fired for the update of this comment.
		assert.Equal(t, hookTaskCount+1, unittest.GetCount(t, &webhook_model.HookTask{}))
		// Issue history was saved for this comment.
		unittest.AssertExistsAndLoadBean(t, &issues_model.ContentHistory{CommentID: comment.ID, IsFirstCreated: true, ContentText: oldContent})
		unittest.AssertExistsAndLoadBean(t, &issues_model.ContentHistory{CommentID: comment.ID, ContentText: comment.Content}, "is_first_created = false")
	})

	t.Run("Comment of pending review", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		comment := unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{ID: 4}, "review_id != 0")
		review := unittest.AssertExistsAndLoadBean(t, &issues_model.Review{ID: comment.ReviewID})
		assert.Equal(t, issues_model.ReviewTypePending, review.Type)
		issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: comment.IssueID})
		unittest.AssertNotExistsBean(t, &issues_model.ContentHistory{CommentID: comment.ID})
		require.NoError(t, webhook_model.CreateWebhook(db.DefaultContext, &webhook_model.Webhook{
			RepoID:   issue.RepoID,
			IsActive: true,
			Events:   `{"choose_events":true,"events":{"issue_comment": true}}`,
		}))
		hookTaskCount := unittest.GetCount(t, &webhook_model.HookTask{})
		oldContent := comment.Content
		comment.Content = "Hello!"

		require.NoError(t, issue_service.UpdateComment(db.DefaultContext, comment, 1, admin, oldContent))

		newComment := unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{ID: 2})
		// Content was updated.
		assert.Equal(t, comment.Content, newComment.Content)
		// Content version was updated.
		assert.Equal(t, 2, newComment.ContentVersion)
		// No notification was fired for the update of this comment.
		assert.Equal(t, hookTaskCount, unittest.GetCount(t, &webhook_model.HookTask{}))
		// Issue history was not saved for this comment.
		unittest.AssertNotExistsBean(t, &issues_model.ContentHistory{CommentID: comment.ID})
	})
}
