// Copyright 2019 The Gitea Authors.
// All rights reserved.
// SPDX-License-Identifier: MIT

package pull

import (
	"context"
	"errors"
	"fmt"

	"forgejo.org/models/db"
	git_model "forgejo.org/models/git"
	issues_model "forgejo.org/models/issues"
	"forgejo.org/modules/git"
	"forgejo.org/modules/gitrepo"
	"forgejo.org/modules/log"
	"forgejo.org/modules/structs"

	"github.com/gobwas/glob"
)

// MergeRequiredContextsCommitStatus returns a commit status state for given required contexts
func MergeRequiredContextsCommitStatus(commitStatuses []*git_model.CommitStatus, requiredContexts []string) structs.CommitStatusState {
	// matchedCount is the number of `CommitStatus.Context` that match any context of `requiredContexts`
	matchedCount := 0
	returnedStatus := structs.CommitStatusSuccess

	if len(requiredContexts) > 0 {
		requiredContextsGlob := make(map[string]glob.Glob, len(requiredContexts))
		for _, ctx := range requiredContexts {
			if gp, err := glob.Compile(ctx); err != nil {
				log.Error("glob.Compile %s failed. Error: %v", ctx, err)
			} else {
				requiredContextsGlob[ctx] = gp
			}
		}

		for _, gp := range requiredContextsGlob {
			var targetStatus structs.CommitStatusState
			for _, commitStatus := range commitStatuses {
				if gp.Match(commitStatus.Context) {
					targetStatus = commitStatus.State
					matchedCount++
					break
				}
			}

			// If required rule not match any action, then it is pending
			if targetStatus == "" {
				if structs.CommitStatusPending.NoBetterThan(returnedStatus) {
					returnedStatus = structs.CommitStatusPending
				}
				break
			}

			if targetStatus.NoBetterThan(returnedStatus) {
				returnedStatus = targetStatus
			}
		}
	}

	if matchedCount == 0 && returnedStatus == structs.CommitStatusSuccess {
		status := git_model.CalcCommitStatus(commitStatuses)
		if status != nil {
			return status.State
		}
		return ""
	}

	return returnedStatus
}

// IsPullCommitStatusPass returns if all required status checks PASS
func IsPullCommitStatusPass(ctx context.Context, pr *issues_model.PullRequest) (bool, error) {
	pb, err := git_model.GetFirstMatchProtectedBranchRule(ctx, pr.BaseRepoID, pr.BaseBranch)
	if err != nil {
		return false, fmt.Errorf("GetFirstMatchProtectedBranchRule: %w", err)
	}
	if pb == nil || !pb.EnableStatusCheck {
		return true, nil
	}

	state, err := GetPullRequestCommitStatusState(ctx, pr)
	if err != nil {
		return false, err
	}
	return state.IsSuccess(), nil
}

// GetPullRequestCommitStatusState returns pull request merged commit status state
func GetPullRequestCommitStatusState(ctx context.Context, pr *issues_model.PullRequest) (structs.CommitStatusState, error) {
	// Ensure HeadRepo is loaded
	if err := pr.LoadHeadRepo(ctx); err != nil {
		return "", fmt.Errorf("LoadHeadRepo: %w", err)
	}

	// check if all required status checks are successful
	headGitRepo, closer, err := gitrepo.RepositoryFromContextOrOpen(ctx, pr.HeadRepo)
	if err != nil {
		return "", fmt.Errorf("RepositoryFromContextOrOpen: %w", err)
	}
	defer closer.Close()

	if pr.Flow == issues_model.PullRequestFlowGithub && !headGitRepo.IsBranchExist(pr.HeadBranch) {
		return "", errors.New("head branch does not exist, can not merge")
	}
	if pr.Flow == issues_model.PullRequestFlowAGit && !git.IsReferenceExist(ctx, headGitRepo.Path, pr.GetGitRefName()) {
		return "", errors.New("head branch does not exist, can not merge")
	}

	var sha string
	if pr.Flow == issues_model.PullRequestFlowGithub {
		sha, err = headGitRepo.GetBranchCommitID(pr.HeadBranch)
	} else {
		sha, err = headGitRepo.GetRefCommitID(pr.GetGitRefName())
	}
	if err != nil {
		return "", err
	}

	if err := pr.LoadBaseRepo(ctx); err != nil {
		return "", fmt.Errorf("LoadBaseRepo: %w", err)
	}

	commitStatuses, _, err := git_model.GetLatestCommitStatus(ctx, pr.BaseRepo.ID, sha, db.ListOptionsAll)
	if err != nil {
		return "", fmt.Errorf("GetLatestCommitStatus: %w", err)
	}

	pb, err := git_model.GetFirstMatchProtectedBranchRule(ctx, pr.BaseRepoID, pr.BaseBranch)
	if err != nil {
		return "", fmt.Errorf("GetFirstMatchProtectedBranchRule: %w", err)
	}
	var requiredContexts []string
	if pb != nil {
		requiredContexts = pb.StatusCheckContexts
	}

	return MergeRequiredContextsCommitStatus(commitStatuses, requiredContexts), nil
}
