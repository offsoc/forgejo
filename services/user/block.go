// Copyright 2023 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT
package user

import (
	"context"

	model "forgejo.org/models"
	"forgejo.org/models/db"
	repo_model "forgejo.org/models/repo"
	user_model "forgejo.org/models/user"

	"xorm.io/builder"
)

// BlockUser adds a blocked user entry for userID to block blockID.
// TODO: Figure out if instance admins should be immune to blocking.
// TODO: Add more mechanism like removing blocked user as collaborator on
// repositories where the user is an owner.
func BlockUser(ctx context.Context, userID, blockID int64) error {
	if userID == blockID || user_model.IsBlocked(ctx, userID, blockID) {
		return nil
	}

	ctx, committer, err := db.TxContext(ctx)
	if err != nil {
		return err
	}
	defer committer.Close()

	// Add the blocked user entry.
	_, err = db.GetEngine(ctx).Insert(&user_model.BlockedUser{UserID: userID, BlockID: blockID})
	if err != nil {
		return err
	}

	// Unfollow the user from the block's perspective.
	err = user_model.UnfollowUser(ctx, blockID, userID)
	if err != nil {
		return err
	}

	// Unfollow the user from the doer's perspective.
	err = user_model.UnfollowUser(ctx, userID, blockID)
	if err != nil {
		return err
	}

	// Blocked user unwatch all repository owned by the doer.
	repoIDs, err := repo_model.GetWatchedRepoIDsOwnedBy(ctx, blockID, userID)
	if err != nil {
		return err
	}

	err = repo_model.UnwatchRepos(ctx, blockID, repoIDs)
	if err != nil {
		return err
	}

	// Remove blocked user as collaborator from repositories the user owns as an
	// individual.
	collabsID, err := repo_model.GetCollaboratorWithUser(ctx, userID, blockID)
	if err != nil {
		return err
	}

	_, err = db.GetEngine(ctx).In("id", collabsID).Delete(&repo_model.Collaboration{})
	if err != nil {
		return err
	}

	// Remove pending repository transfers, and set the status on those repository
	// back to ready.
	pendingTransfersIDs, err := model.GetPendingTransferIDs(ctx, userID, blockID)
	if err != nil {
		return err
	}

	// Use a subquery instead of a JOIN, because not every database supports JOIN
	// on a UPDATE query.
	_, err = db.GetEngine(ctx).Table("repository").
		In("id", builder.Select("repo_id").From("repo_transfer").Where(builder.In("id", pendingTransfersIDs))).
		Cols("status").
		Update(&repo_model.Repository{Status: repo_model.RepositoryReady})
	if err != nil {
		return err
	}

	_, err = db.GetEngine(ctx).In("id", pendingTransfersIDs).Delete(&model.RepoTransfer{})
	if err != nil {
		return err
	}

	return committer.Commit()
}
