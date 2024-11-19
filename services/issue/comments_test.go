// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issue_test

import (
	"testing"

	"code.gitea.io/gitea/models/db"
	issues_model "code.gitea.io/gitea/models/issues"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	webhook_model "code.gitea.io/gitea/models/webhook"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/test"
	issue_service "code.gitea.io/gitea/services/issue"
	"code.gitea.io/gitea/tests"

	_ "code.gitea.io/gitea/services/webhook"

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
		assert.EqualValues(t, issue.NumComments-1, unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: comment.IssueID}).NumComments)
		// A notification was fired for the deletion of this comment.
		assert.EqualValues(t, hookTaskCount+1, unittest.GetCount(t, &webhook_model.HookTask{}))
	})

	t.Run("Comment of pending review", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		// We have to ensure that this comment's linked review is pending.
		comment := unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{ID: 4}, "review_id != 0")
		review := unittest.AssertExistsAndLoadBean(t, &issues_model.Review{ID: comment.ReviewID})
		assert.EqualValues(t, issues_model.ReviewTypePending, review.Type)
		issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: comment.IssueID})

		require.NoError(t, webhook_model.CreateWebhook(db.DefaultContext, &webhook_model.Webhook{
			RepoID:   issue.RepoID,
			IsActive: true,
			Events:   `{"choose_events":true,"events":{"issue_comment": true}}`,
		}))
		hookTaskCount := unittest.GetCount(t, &webhook_model.HookTask{})

		require.NoError(t, issue_service.DeleteComment(db.DefaultContext, nil, comment))

		// The comment doesn't exist anymore.
		unittest.AssertNotExistsBean(t, &issues_model.Comment{ID: comment.ID})
		// Ensure that the number of comments wasn't decreased.
		assert.EqualValues(t, issue.NumComments, unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: comment.IssueID}).NumComments)
		// No notification was fired for the deletion of this comment.
		assert.EqualValues(t, hookTaskCount, unittest.GetCount(t, &webhook_model.HookTask{}))
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
		assert.EqualValues(t, comment.Content, newComment.Content)
		// Content version was updated.
		assert.EqualValues(t, 2, newComment.ContentVersion)
		// A notification was fired for the update of this comment.
		assert.EqualValues(t, hookTaskCount+1, unittest.GetCount(t, &webhook_model.HookTask{}))
		// Issue history was saved for this comment.
		unittest.AssertExistsAndLoadBean(t, &issues_model.ContentHistory{CommentID: comment.ID, IsFirstCreated: true, ContentText: oldContent})
		unittest.AssertExistsAndLoadBean(t, &issues_model.ContentHistory{CommentID: comment.ID, ContentText: comment.Content}, "is_first_created = false")
	})

	t.Run("Comment of pending review", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		comment := unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{ID: 4}, "review_id != 0")
		review := unittest.AssertExistsAndLoadBean(t, &issues_model.Review{ID: comment.ReviewID})
		assert.EqualValues(t, issues_model.ReviewTypePending, review.Type)
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
		assert.EqualValues(t, comment.Content, newComment.Content)
		// Content version was updated.
		assert.EqualValues(t, 2, newComment.ContentVersion)
		// No notification was fired for the update of this comment.
		assert.EqualValues(t, hookTaskCount, unittest.GetCount(t, &webhook_model.HookTask{}))
		// Issue history was not saved for this comment.
		unittest.AssertNotExistsBean(t, &issues_model.ContentHistory{CommentID: comment.ID})
	})
}
