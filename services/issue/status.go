// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issue

import (
	"context"

	issues_model "forgejo.org/models/issues"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/log"
	notify_service "forgejo.org/services/notify"
)

// ChangeStatus changes issue status to open or closed.
func ChangeStatus(ctx context.Context, issue *issues_model.Issue, doer *user_model.User, commitID string, closed bool) error {
	comment, err := issues_model.ChangeIssueStatus(ctx, issue, doer, closed)
	if err != nil {
		if issues_model.IsErrDependenciesLeft(err) && closed {
			if err := issues_model.FinishIssueStopwatchIfPossible(ctx, doer, issue); err != nil {
				log.Error("Unable to stop stopwatch for issue[%d]#%d: %v", issue.ID, issue.Index, err)
			}
		}
		return err
	}

	if closed {
		if err := issues_model.FinishIssueStopwatchIfPossible(ctx, doer, issue); err != nil {
			return err
		}
	}

	notify_service.IssueChangeStatus(ctx, doer, commitID, issue, comment, closed)

	return nil
}
