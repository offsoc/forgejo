// Copyright 2024 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package automerge

import (
	"context"

	issues_model "forgejo.org/models/issues"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/log"
	notify_service "forgejo.org/services/notify"
)

type automergeNotifier struct {
	notify_service.NullNotifier
}

var _ notify_service.Notifier = &automergeNotifier{}

// NewNotifier create a new automergeNotifier notifier
func NewNotifier() notify_service.Notifier {
	return &automergeNotifier{}
}

func (n *automergeNotifier) PullRequestReview(ctx context.Context, pr *issues_model.PullRequest, review *issues_model.Review, comment *issues_model.Comment, mentions []*user_model.User) {
	// as a missing / blocking reviews could have blocked a pending automerge let's recheck
	if review.Type == issues_model.ReviewTypeApprove {
		if err := StartPRCheckAndAutoMergeBySHA(ctx, review.CommitID, pr.BaseRepo); err != nil {
			log.Error("StartPullRequestAutoMergeCheckBySHA: %v", err)
		}
	}
}

func (n *automergeNotifier) PullReviewDismiss(ctx context.Context, doer *user_model.User, review *issues_model.Review, comment *issues_model.Comment) {
	if err := review.LoadIssue(ctx); err != nil {
		log.Error("LoadIssue: %v", err)
		return
	}
	if err := review.Issue.LoadPullRequest(ctx); err != nil {
		log.Error("LoadPullRequest: %v", err)
		return
	}
	// as reviews could have blocked a pending automerge let's recheck
	StartPRCheckAndAutoMerge(ctx, review.Issue.PullRequest)
}
