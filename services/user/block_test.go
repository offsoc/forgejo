// Copyright 2023 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user

import (
	"testing"

	model "code.gitea.io/gitea/models"
	"code.gitea.io/gitea/models/db"
	issues_model "code.gitea.io/gitea/models/issues"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBlockUser will ensure that when you block a user, certain actions have
// been taken, like unfollowing each other etc.
func TestBlockUser(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 5})
	blockedUser := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})

	t.Run("Follow", func(t *testing.T) {
		defer user_model.UnblockUser(db.DefaultContext, doer.ID, blockedUser.ID)

		// Follow each other.
		require.NoError(t, user_model.FollowUser(db.DefaultContext, doer.ID, blockedUser.ID))
		require.NoError(t, user_model.FollowUser(db.DefaultContext, blockedUser.ID, doer.ID))

		require.NoError(t, BlockUser(db.DefaultContext, doer.ID, blockedUser.ID))

		// Ensure they aren't following each other anymore.
		assert.False(t, user_model.IsFollowing(db.DefaultContext, doer.ID, blockedUser.ID))
		assert.False(t, user_model.IsFollowing(db.DefaultContext, blockedUser.ID, doer.ID))
	})

	t.Run("Watch", func(t *testing.T) {
		defer user_model.UnblockUser(db.DefaultContext, doer.ID, blockedUser.ID)

		// Blocked user watch repository of doer.
		repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{OwnerID: doer.ID})
		require.NoError(t, repo_model.WatchRepo(db.DefaultContext, blockedUser.ID, repo.ID, true))

		require.NoError(t, BlockUser(db.DefaultContext, doer.ID, blockedUser.ID))

		// Ensure blocked user isn't following doer's repository.
		assert.False(t, repo_model.IsWatching(db.DefaultContext, blockedUser.ID, repo.ID))
	})

	t.Run("Collaboration", func(t *testing.T) {
		doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 16})
		blockedUser := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 18})
		repo1 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 22, OwnerID: doer.ID})
		repo2 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 21, OwnerID: doer.ID})
		defer user_model.UnblockUser(db.DefaultContext, doer.ID, blockedUser.ID)

		isBlockedUserCollab := func(repo *repo_model.Repository) bool {
			isCollaborator, err := repo_model.IsCollaborator(db.DefaultContext, repo.ID, blockedUser.ID)
			require.NoError(t, err)
			return isCollaborator
		}

		assert.True(t, isBlockedUserCollab(repo1))
		assert.True(t, isBlockedUserCollab(repo2))

		require.NoError(t, BlockUser(db.DefaultContext, doer.ID, blockedUser.ID))

		assert.False(t, isBlockedUserCollab(repo1))
		assert.False(t, isBlockedUserCollab(repo2))
	})

	t.Run("Pending transfers", func(t *testing.T) {
		doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
		blockedUser := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 3})
		defer user_model.UnblockUser(db.DefaultContext, doer.ID, blockedUser.ID)

		unittest.AssertExistsIf(t, true, &repo_model.Repository{ID: 3, OwnerID: blockedUser.ID, Status: repo_model.RepositoryPendingTransfer})
		unittest.AssertExistsIf(t, true, &model.RepoTransfer{ID: 1, RecipientID: doer.ID, DoerID: blockedUser.ID})

		require.NoError(t, BlockUser(db.DefaultContext, doer.ID, blockedUser.ID))

		unittest.AssertExistsIf(t, false, &model.RepoTransfer{ID: 1, RecipientID: doer.ID, DoerID: blockedUser.ID})

		// Don't use AssertExistsIf, as it doesn't include the zero values in the condition such as `repo_model.RepositoryReady`.
		repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 3, OwnerID: blockedUser.ID})
		assert.Equal(t, repo_model.RepositoryReady, repo.Status)
	})

	t.Run("Issues", func(t *testing.T) {
		doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
		blockedUser := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 4})
		defer user_model.UnblockUser(db.DefaultContext, doer.ID, blockedUser.ID)

		repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 2, OwnerID: doer.ID})
		issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 4, RepoID: repo.ID}, "is_closed = true")

		_, err := issues_model.ChangeIssueStatus(db.DefaultContext, issue, blockedUser, false)
		require.NoError(t, err)

		_, err = issues_model.ChangeIssueStatus(db.DefaultContext, issue, doer, true)
		require.NoError(t, err)

		require.NoError(t, BlockUser(db.DefaultContext, doer.ID, blockedUser.ID))

		_, err = issues_model.ChangeIssueStatus(db.DefaultContext, issue, blockedUser, false)
		require.Error(t, err)
	})
}
