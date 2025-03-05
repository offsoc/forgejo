// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package moderation

import (
	"errors"

	"code.gitea.io/gitea/models/issues"
	"code.gitea.io/gitea/models/moderation"
	"code.gitea.io/gitea/models/perm"
	access_model "code.gitea.io/gitea/models/perm/access"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/unit"
	"code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/services/context"
)

var (
	ErrContentDoesNotExist = errors.New("the content to be reported does not exist")
	ErrDoerNotAllowed      = errors.New("doer not allowed to access the content to be reported")
)

// CanReport checks if doer has access to the content they are reporting (repository, issue, pull request or comment).
// When reporting repositories the user should have at least read access to any repo unit type.
// When reporting issues, pull requests or comments the user should have at least read access
// to 'TypeIssues', respectively 'TypePullRequests' unit for the repository where the content belongs.
// When reporting users or organizations no checks are made.
func CanReport(ctx context.Context, doer *user.User, contentType moderation.ReportedContentType, contentID int64) (bool, error) {
	var hasAccess bool = false
	var issueID int64 = 0
	var repoID int64 = 0
	var unitType unit.Type = unit.TypeInvalid

	if contentType == moderation.ReportedContentTypeComment {
		comment, err := issues.GetCommentByID(ctx, contentID)
		if err != nil {
			if issues.IsErrCommentNotExist(err) {
				log.Warn("User #%d wanted to report comment #%d but it does not exist.", doer.ID, contentID)
				return false, ErrContentDoesNotExist
			}
			return false, err
		}
		issueID = comment.IssueID
	} else if contentType == moderation.ReportedContentTypeIssue {
		issueID = contentID
	} else if contentType == moderation.ReportedContentTypeRepository {
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
			hasAccess, err = access_model.HasAccessUnit(ctx, doer, repo, unitType, perm.AccessModeRead)
			if err != nil {
				return false, err
			} else if !hasAccess {
				log.Warn("User #%d wanted to report issue #%d or one of its comments from repository #%d but they don't have access to it.", doer.ID, issueID, repoID)
				return false, ErrDoerNotAllowed
			}
		} else {
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

	return hasAccess, nil
}
