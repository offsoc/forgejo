// Copyright 2023 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT
package issue

import (
	"context"

	issues_model "forgejo.org/models/issues"
	user_model "forgejo.org/models/user"
)

// CreateIssueReaction creates a reaction on issue.
func CreateIssueReaction(ctx context.Context, doer *user_model.User, issue *issues_model.Issue, content string) (*issues_model.Reaction, error) {
	if err := issue.LoadRepo(ctx); err != nil {
		return nil, err
	}

	// Check if the doer is blocked by the issue's poster or repository owner.
	if user_model.IsBlockedMultiple(ctx, []int64{issue.PosterID, issue.Repo.OwnerID}, doer.ID) {
		return nil, user_model.ErrBlockedByUser
	}

	return issues_model.CreateReaction(ctx, &issues_model.ReactionOptions{
		Type:    content,
		DoerID:  doer.ID,
		IssueID: issue.ID,
	})
}

// CreateCommentReaction creates a reaction on comment.
func CreateCommentReaction(ctx context.Context, doer *user_model.User, issue *issues_model.Issue, comment *issues_model.Comment, content string) (*issues_model.Reaction, error) {
	if err := issue.LoadRepo(ctx); err != nil {
		return nil, err
	}

	// Check if the doer is blocked by the issue's poster, the comment's poster or repository owner.
	if user_model.IsBlockedMultiple(ctx, []int64{comment.PosterID, issue.PosterID, issue.Repo.OwnerID}, doer.ID) {
		return nil, user_model.ErrBlockedByUser
	}

	return issues_model.CreateReaction(ctx, &issues_model.ReactionOptions{
		Type:      content,
		DoerID:    doer.ID,
		IssueID:   issue.ID,
		CommentID: comment.ID,
	})
}
