// Copyright 2019 The Gitea Authors. All rights reserved.
// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package feed

import (
	"strings"
	"testing"

	activities_model "forgejo.org/models/activities"
	"forgejo.org/models/db"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/git"
	"forgejo.org/modules/repository"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/test"

	_ "forgejo.org/models/actions"
	_ "forgejo.org/models/forgefed"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	unittest.MainTest(m)
}

func TestRenameRepoAction(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{OwnerID: user.ID})
	repo.Owner = user

	oldRepoName := repo.Name
	const newRepoName = "newRepoName"
	repo.Name = newRepoName
	repo.LowerName = strings.ToLower(newRepoName)

	actionBean := &activities_model.Action{
		OpType:    activities_model.ActionRenameRepo,
		ActUserID: user.ID,
		ActUser:   user,
		RepoID:    repo.ID,
		Repo:      repo,
		IsPrivate: repo.IsPrivate,
		Content:   oldRepoName,
	}
	unittest.AssertNotExistsBean(t, actionBean)

	NewNotifier().RenameRepository(db.DefaultContext, user, repo, oldRepoName)

	unittest.AssertExistsAndLoadBean(t, actionBean)
	unittest.CheckConsistencyFor(t, &activities_model.Action{})
}

func pushCommits() *repository.PushCommits {
	pushCommits := repository.NewPushCommits()
	pushCommits.Commits = []*repository.PushCommit{
		{
			Sha1:           "69554a6",
			CommitterEmail: "user2@example.com",
			CommitterName:  "User2",
			AuthorEmail:    "user2@example.com",
			AuthorName:     "User2",
			Message:        "not signed commit",
		},
		{
			Sha1:           "27566bd",
			CommitterEmail: "user2@example.com",
			CommitterName:  "User2",
			AuthorEmail:    "user2@example.com",
			AuthorName:     "User2",
			Message:        "good signed commit (with not yet validated email)",
		},
		{
			Sha1:           "5099b81",
			CommitterEmail: "user2@example.com",
			CommitterName:  "User2",
			AuthorEmail:    "user2@example.com",
			AuthorName:     "User2",
			Message:        "good signed commit",
		},
	}
	pushCommits.HeadCommit = &repository.PushCommit{Sha1: "69554a6"}
	return pushCommits
}

func TestSyncPushCommits(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{OwnerID: user.ID})

	t.Run("All commits", func(t *testing.T) {
		defer test.MockVariableValue(&setting.UI.FeedMaxCommitNum, 10)()

		maxID := unittest.GetCount(t, &activities_model.Action{})
		NewNotifier().SyncPushCommits(db.DefaultContext, user, repo, &repository.PushUpdateOptions{RefFullName: git.RefNameFromBranch("master")}, pushCommits())

		newNotification := unittest.AssertExistsAndLoadBean(t, &activities_model.Action{ActUserID: user.ID, RefName: "refs/heads/master"}, unittest.Cond("id > ?", maxID))
		assert.JSONEq(t, `{"Commits":[{"Sha1":"69554a6","Message":"not signed commit","AuthorEmail":"user2@example.com","AuthorName":"User2","CommitterEmail":"user2@example.com","CommitterName":"User2","Signature":null,"Verification":null,"Timestamp":"0001-01-01T00:00:00Z"},{"Sha1":"27566bd","Message":"good signed commit (with not yet validated email)","AuthorEmail":"user2@example.com","AuthorName":"User2","CommitterEmail":"user2@example.com","CommitterName":"User2","Signature":null,"Verification":null,"Timestamp":"0001-01-01T00:00:00Z"},{"Sha1":"5099b81","Message":"good signed commit","AuthorEmail":"user2@example.com","AuthorName":"User2","CommitterEmail":"user2@example.com","CommitterName":"User2","Signature":null,"Verification":null,"Timestamp":"0001-01-01T00:00:00Z"}],"HeadCommit":{"Sha1":"69554a6","Message":"","AuthorEmail":"","AuthorName":"","CommitterEmail":"","CommitterName":"","Signature":null,"Verification":null,"Timestamp":"0001-01-01T00:00:00Z"},"CompareURL":"","Len":0}`, newNotification.Content)
	})

	t.Run("Only one commit", func(t *testing.T) {
		defer test.MockVariableValue(&setting.UI.FeedMaxCommitNum, 1)()

		maxID := unittest.GetCount(t, &activities_model.Action{})
		NewNotifier().SyncPushCommits(db.DefaultContext, user, repo, &repository.PushUpdateOptions{RefFullName: git.RefNameFromBranch("main")}, pushCommits())

		newNotification := unittest.AssertExistsAndLoadBean(t, &activities_model.Action{ActUserID: user.ID, RefName: "refs/heads/main"}, unittest.Cond("id > ?", maxID))
		assert.JSONEq(t, `{"Commits":[{"Sha1":"69554a6","Message":"not signed commit","AuthorEmail":"user2@example.com","AuthorName":"User2","CommitterEmail":"user2@example.com","CommitterName":"User2","Signature":null,"Verification":null,"Timestamp":"0001-01-01T00:00:00Z"}],"HeadCommit":{"Sha1":"69554a6","Message":"","AuthorEmail":"","AuthorName":"","CommitterEmail":"","CommitterName":"","Signature":null,"Verification":null,"Timestamp":"0001-01-01T00:00:00Z"},"CompareURL":"","Len":0}`, newNotification.Content)
	})
}

func TestPushCommits(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{OwnerID: user.ID})

	t.Run("All commits", func(t *testing.T) {
		defer test.MockVariableValue(&setting.UI.FeedMaxCommitNum, 10)()

		maxID := unittest.GetCount(t, &activities_model.Action{})
		NewNotifier().PushCommits(db.DefaultContext, user, repo, &repository.PushUpdateOptions{RefFullName: git.RefNameFromBranch("master")}, pushCommits())

		newNotification := unittest.AssertExistsAndLoadBean(t, &activities_model.Action{ActUserID: user.ID, RefName: "refs/heads/master"}, unittest.Cond("id > ?", maxID))
		assert.JSONEq(t, `{"Commits":[{"Sha1":"69554a6","Message":"not signed commit","AuthorEmail":"user2@example.com","AuthorName":"User2","CommitterEmail":"user2@example.com","CommitterName":"User2","Signature":null,"Verification":null,"Timestamp":"0001-01-01T00:00:00Z"},{"Sha1":"27566bd","Message":"good signed commit (with not yet validated email)","AuthorEmail":"user2@example.com","AuthorName":"User2","CommitterEmail":"user2@example.com","CommitterName":"User2","Signature":null,"Verification":null,"Timestamp":"0001-01-01T00:00:00Z"},{"Sha1":"5099b81","Message":"good signed commit","AuthorEmail":"user2@example.com","AuthorName":"User2","CommitterEmail":"user2@example.com","CommitterName":"User2","Signature":null,"Verification":null,"Timestamp":"0001-01-01T00:00:00Z"}],"HeadCommit":{"Sha1":"69554a6","Message":"","AuthorEmail":"","AuthorName":"","CommitterEmail":"","CommitterName":"","Signature":null,"Verification":null,"Timestamp":"0001-01-01T00:00:00Z"},"CompareURL":"","Len":0}`, newNotification.Content)
	})

	t.Run("Only one commit", func(t *testing.T) {
		defer test.MockVariableValue(&setting.UI.FeedMaxCommitNum, 1)()

		maxID := unittest.GetCount(t, &activities_model.Action{})
		NewNotifier().PushCommits(db.DefaultContext, user, repo, &repository.PushUpdateOptions{RefFullName: git.RefNameFromBranch("main")}, pushCommits())

		newNotification := unittest.AssertExistsAndLoadBean(t, &activities_model.Action{ActUserID: user.ID, RefName: "refs/heads/main"}, unittest.Cond("id > ?", maxID))
		assert.JSONEq(t, `{"Commits":[{"Sha1":"69554a6","Message":"not signed commit","AuthorEmail":"user2@example.com","AuthorName":"User2","CommitterEmail":"user2@example.com","CommitterName":"User2","Signature":null,"Verification":null,"Timestamp":"0001-01-01T00:00:00Z"}],"HeadCommit":{"Sha1":"69554a6","Message":"","AuthorEmail":"","AuthorName":"","CommitterEmail":"","CommitterName":"","Signature":null,"Verification":null,"Timestamp":"0001-01-01T00:00:00Z"},"CompareURL":"","Len":0}`, newNotification.Content)
	})
}
