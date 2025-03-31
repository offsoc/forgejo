// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package moderation

import (
	"errors"

	"forgejo.org/models/issues"
	"forgejo.org/models/moderation"
	"forgejo.org/models/perm"
	access_model "forgejo.org/models/perm/access"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unit"
	"forgejo.org/models/user"
	"forgejo.org/modules/log"
	"forgejo.org/services/context"
)

var (
	ErrContentDoesNotExist = errors.New("the content to be reported does not exist")
	ErrDoerNotAllowed      = errors.New("doer not allowed to access the content to be reported")
)

// CanReport checks if doer has access to the content they are reporting
// (user, organization, repository, issue, pull request or comment).
// When reporting repositories the user should have at least read access to any repo unit type.
// When reporting issues, pull requests or comments the user should have at least read access
// to 'TypeIssues', respectively 'TypePullRequests' unit for the repository where the content belongs.
// When reporting users or organizations doer should be able to view the reported entity.
func CanReport(ctx context.Context, doer *user.User, contentType moderation.ReportedContentType, contentID int64) (bool, error) {
	hasAccess := false
	var issueID int64
	var repoID int64
	unitType := unit.TypeInvalid // used when checking access for issues, pull requests or comments

	if contentType == moderation.ReportedContentTypeUser {
		reportedUser, err := user.GetUserByID(ctx, contentID)
		if err != nil {
			if user.IsErrUserNotExist(err) {
				log.Warn("User #%d wanted to report user #%d but it does not exist.", doer.ID, contentID)
				return false, ErrContentDoesNotExist
			}
			return false, err
		}

		hasAccess = user.IsUserVisibleToViewer(ctx, reportedUser, ctx.Doer)
		if !hasAccess {
			log.Warn("User #%d wanted to report user/org #%d but they are not able to see that profile.", doer.ID, contentID)
			return false, ErrDoerNotAllowed
		}
	} else {
		// for comments and issues/pulls we need to get the parent repository
		switch contentType {
		case moderation.ReportedContentTypeComment:
			comment, err := issues.GetCommentByID(ctx, contentID)
			if err != nil {
				if issues.IsErrCommentNotExist(err) {
					log.Warn("User #%d wanted to report comment #%d but it does not exist.", doer.ID, contentID)
					return false, ErrContentDoesNotExist
				}
				return false, err
			}
			if !comment.Type.HasContentSupport() {
				// this is not a comment with text and/or attachments
				log.Warn("User #%d wanted to report comment #%d but it is not a comment with content.", doer.ID, contentID)
				return false, nil
			}
			issueID = comment.IssueID
		case moderation.ReportedContentTypeIssue:
			issueID = contentID
		case moderation.ReportedContentTypeRepository:
			repoID = contentID
		}

		if issueID > 0 {
			issue, err := issues.GetIssueByID(ctx, issueID)
			if err != nil {
				if issues.IsErrIssueNotExist(err) {
					log.Warn("User #%d wanted to report issue #%d (or one of its comments) but it does not exist.", doer.ID, issueID)
					return false, ErrContentDoesNotExist
				}
				return false, err
			}

			repoID = issue.RepoID
			if issue.IsPull {
				unitType = unit.TypePullRequests
			} else {
				unitType = unit.TypeIssues
			}
		}

		if repoID > 0 {
			repo, err := repo_model.GetRepositoryByID(ctx, repoID)
			if err != nil {
				if repo_model.IsErrRepoNotExist(err) {
					log.Warn("User #%d wanted to report repository #%d (or one of its issues / comments) but it does not exist.", doer.ID, repoID)
					return false, ErrContentDoesNotExist
				}
				return false, err
			}

			if issueID > 0 {
				// for comments and issues/pulls doer should have at least read access to the corresponding repo unit (issues, respectively pull requests)
				hasAccess, err = access_model.HasAccessUnit(ctx, doer, repo, unitType, perm.AccessModeRead)
				if err != nil {
					return false, err
				} else if !hasAccess {
					log.Warn("User #%d wanted to report issue #%d or one of its comments from repository #%d but they don't have access to it.", doer.ID, issueID, repoID)
					return false, ErrDoerNotAllowed
				}
			} else {
				// for repositories doer should have at least read access to at least one repo unit
				perm, err := access_model.GetUserRepoPermission(ctx, repo, doer)
				if err != nil {
					return false, err
				}
				hasAccess = perm.CanReadAny(unit.AllRepoUnitTypes...)
				if !hasAccess {
					log.Warn("User #%d wanted to report repository #%d but they don't have access to it.", doer.ID, repoID)
					return false, ErrDoerNotAllowed
				}
			}
		}
	}

	return hasAccess, nil
}
