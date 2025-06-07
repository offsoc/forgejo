// Copyright 2025 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issue

import (
	"context"

	issues_model "forgejo.org/models/issues"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/modules/optional"
	"forgejo.org/modules/structs"
)

func GetSuggestions(ctx context.Context, repo *repo_model.Repository, isPull optional.Option[bool]) ([]*structs.IssueSuggestion, error) {
	var issues issues_model.IssueList
	var err error
	pageSize := 1000

	issues, err = issues_model.FindLatestUpdatedIssues(ctx, repo.ID, isPull, pageSize)
	if err != nil {
		return nil, err
	}

	if err := issues.LoadPullRequests(ctx); err != nil {
		return nil, err
	}

	suggestions := make([]*structs.IssueSuggestion, 0, len(issues))
	for _, issue := range issues {
		suggestion := &structs.IssueSuggestion{
			Index: issue.Index,
			Title: issue.Title,
			State: issue.State(),
		}

		if issue.IsPull && issue.PullRequest != nil {
			suggestion.IsPr = true
			suggestion.HasMerged = issue.PullRequest.HasMerged
			suggestion.IsWorkInProgress = issue.PullRequest.IsWorkInProgress(ctx)
		}
		suggestions = append(suggestions, suggestion)
	}

	return suggestions, nil
}
