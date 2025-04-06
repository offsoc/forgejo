// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package agit

import (
	"context"
	"fmt"
	"os"
	"strings"

	issues_model "forgejo.org/models/issues"
	repo_model "forgejo.org/models/repo"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/git"
	"forgejo.org/modules/git/pushoptions"
	"forgejo.org/modules/log"
	"forgejo.org/modules/private"
	notify_service "forgejo.org/services/notify"
	pull_service "forgejo.org/services/pull"
)

// ProcReceive handle proc receive work
func ProcReceive(ctx context.Context, repo *repo_model.Repository, gitRepo *git.Repository, opts *private.HookOptions) ([]private.HookProcReceiveRefResult, error) {
	results := make([]private.HookProcReceiveRefResult, 0, len(opts.OldCommitIDs))

	topicBranch, _ := opts.GetGitPushOptions().GetString(pushoptions.AgitTopic)
	_, forcePush := opts.GetGitPushOptions().GetString(pushoptions.AgitForcePush)
	title, hasTitle := opts.GetGitPushOptions().GetString(pushoptions.AgitTitle)
	description, hasDesc := opts.GetGitPushOptions().GetString(pushoptions.AgitDescription)

	objectFormat := git.ObjectFormatFromName(repo.ObjectFormatName)

	pusher, err := user_model.GetUserByID(ctx, opts.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user[%d]: %w", opts.UserID, err)
	}

	for i := range opts.OldCommitIDs {
		// Avoid processing this change if the new commit is empty.
		if opts.NewCommitIDs[i] == objectFormat.EmptyObjectID().String() {
			results = append(results, private.HookProcReceiveRefResult{
				OriginalRef: opts.RefFullNames[i],
				OldOID:      opts.OldCommitIDs[i],
				NewOID:      opts.NewCommitIDs[i],
				Err:         "Cannot delete a non-existent branch.",
			})
			continue
		}

		// Only process references that are in the form of refs/for/
		if !opts.RefFullNames[i].IsFor() {
			results = append(results, private.HookProcReceiveRefResult{
				IsNotMatched: true,
				OriginalRef:  opts.RefFullNames[i],
			})
			continue
		}

		// Get the anything after the refs/for/ prefix.
		baseBranchName := opts.RefFullNames[i].ForBranchName()
		curentTopicBranch := topicBranch

		// If the reference was given in the format of refs/for/<target-branch>/<topic-branch>,
		// where <target-branch> and <topic-branch> can contain slashes, we need to iteratively
		// search for what the target and topic branch is.
		if !gitRepo.IsBranchExist(baseBranchName) {
			for p, v := range baseBranchName {
				if v == '/' && gitRepo.IsBranchExist(baseBranchName[:p]) && p != len(baseBranchName)-1 {
					curentTopicBranch = baseBranchName[p+1:]
					baseBranchName = baseBranchName[:p]
					break
				}
			}
		}

		if len(curentTopicBranch) == 0 {
			results = append(results, private.HookProcReceiveRefResult{
				OriginalRef: opts.RefFullNames[i],
				OldOID:      opts.OldCommitIDs[i],
				NewOID:      opts.NewCommitIDs[i],
				Err:         "The topic-branch option is not set",
			})
			continue
		}

		// Include the user's name in the head branch, to avoid conflicts
		// with other users.
		headBranch := curentTopicBranch
		userName := strings.ToLower(opts.UserName)
		if !strings.HasPrefix(curentTopicBranch, userName+"/") {
			headBranch = userName + "/" + curentTopicBranch
		}

		// Check if a AGit pull request already exist for this branch.
		pr, err := issues_model.GetUnmergedPullRequest(ctx, repo.ID, repo.ID, headBranch, baseBranchName, issues_model.PullRequestFlowAGit)
		if err != nil {
			if !issues_model.IsErrPullRequestNotExist(err) {
				return nil, fmt.Errorf("failed to get unmerged AGit flow pull request in repository %q: %w", repo.FullName(), err)
			}

			// Check if the changes are already in the target branch.
			stdout, _, gitErr := git.NewCommand(ctx, "branch", "--contains").AddDynamicArguments(opts.NewCommitIDs[i], baseBranchName).RunStdString(&git.RunOpts{Dir: repo.RepoPath()})
			if gitErr != nil {
				return nil, fmt.Errorf("failed to check if the target branch already contains the new commit in repository %q: %w", repo.FullName(), err)
			}
			if len(stdout) > 0 {
				results = append(results, private.HookProcReceiveRefResult{
					OriginalRef: opts.RefFullNames[i],
					OldOID:      opts.OldCommitIDs[i],
					NewOID:      opts.NewCommitIDs[i],
					Err:         "The target branch already contains this commit",
				})
				continue
			}

			// Automatically fill out the title and the description from the first commit.
			shouldGetCommit := len(title) == 0 || len(description) == 0

			var commit *git.Commit
			if shouldGetCommit {
				commit, err = gitRepo.GetCommit(opts.NewCommitIDs[i])
				if err != nil {
					return nil, fmt.Errorf("failed to get commit %s in repository %q: %w", opts.NewCommitIDs[i], repo.FullName(), err)
				}
			}
			if !hasTitle || len(title) == 0 {
				title = strings.Split(commit.CommitMessage, "\n")[0]
			}
			if !hasDesc || len(description) == 0 {
				_, description, _ = strings.Cut(commit.CommitMessage, "\n\n")
			}

			prIssue := &issues_model.Issue{
				RepoID:   repo.ID,
				Title:    title,
				PosterID: pusher.ID,
				Poster:   pusher,
				IsPull:   true,
				Content:  description,
			}

			pr := &issues_model.PullRequest{
				HeadRepoID:   repo.ID,
				BaseRepoID:   repo.ID,
				HeadBranch:   headBranch,
				HeadCommitID: opts.NewCommitIDs[i],
				BaseBranch:   baseBranchName,
				HeadRepo:     repo,
				BaseRepo:     repo,
				MergeBase:    "",
				Type:         issues_model.PullRequestGitea,
				Flow:         issues_model.PullRequestFlowAGit,
			}

			if err := pull_service.NewPullRequest(ctx, repo, prIssue, []int64{}, []string{}, pr, []int64{}); err != nil {
				return nil, fmt.Errorf("unable to create new pull request: %w", err)
			}

			log.Trace("Pull request created: %d/%d", repo.ID, prIssue.ID)

			results = append(results, private.HookProcReceiveRefResult{
				Ref:         pr.GetGitRefName(),
				OriginalRef: opts.RefFullNames[i],
				OldOID:      objectFormat.EmptyObjectID().String(),
				NewOID:      opts.NewCommitIDs[i],
			})
			continue
		}

		// Update an existing pull request.
		if err := pr.LoadBaseRepo(ctx); err != nil {
			return nil, fmt.Errorf("unable to load base repository for PR[%d]: %w", pr.ID, err)
		}

		oldCommitID, err := gitRepo.GetRefCommitID(pr.GetGitRefName())
		if err != nil {
			return nil, fmt.Errorf("unable to get commit id of reference[%s] in base repository for PR[%d]: %w", pr.GetGitRefName(), pr.ID, err)
		}

		// Do not process this change if nothing was changed.
		if oldCommitID == opts.NewCommitIDs[i] {
			results = append(results, private.HookProcReceiveRefResult{
				OriginalRef: opts.RefFullNames[i],
				OldOID:      opts.OldCommitIDs[i],
				NewOID:      opts.NewCommitIDs[i],
				Err:         "The new commit is the same as the old commit",
			})
			continue
		}

		// If the force push option was not set, ensure that this change isn't a force push.
		if !forcePush {
			output, _, err := git.NewCommand(ctx, "rev-list", "--max-count=1").AddDynamicArguments(oldCommitID, "^"+opts.NewCommitIDs[i]).RunStdString(&git.RunOpts{Dir: repo.RepoPath(), Env: os.Environ()})
			if err != nil {
				return nil, fmt.Errorf("failed to detect a force push: %w", err)
			} else if len(output) > 0 {
				results = append(results, private.HookProcReceiveRefResult{
					OriginalRef: opts.RefFullNames[i],
					OldOID:      opts.OldCommitIDs[i],
					NewOID:      opts.NewCommitIDs[i],
					Err:         "Updates were rejected because the tip of your current branch is behind its remote counterpart. If this is intentional, set the `force-push` option by adding `-o force-push=true` to your `git push` command.",
				})
				continue
			}
		}

		// Set the new commit as reference of the pull request.
		pr.HeadCommitID = opts.NewCommitIDs[i]
		if err = pull_service.UpdateRef(ctx, pr); err != nil {
			return nil, fmt.Errorf("failed to update the reference of the pull request: %w", err)
		}

		// TODO: refactor to unify with `pull_service.AddTestPullRequestTask`

		// Add the pull request to the merge conflicting checker queue.
		pull_service.AddToTaskQueue(ctx, pr)

		if err := pr.LoadIssue(ctx); err != nil {
			return nil, fmt.Errorf("failed to load the issue of the pull request: %w", err)
		}

		// Validate pull request.
		pull_service.ValidatePullRequest(ctx, pr, oldCommitID, opts.NewCommitIDs[i], pusher)

		// TODO: call `InvalidateCodeComments`

		// Create and notify about the new commits.
		comment, err := pull_service.CreatePushPullComment(ctx, pusher, pr, oldCommitID, opts.NewCommitIDs[i])
		if err == nil && comment != nil {
			notify_service.PullRequestPushCommits(ctx, pusher, pr, comment)
		}
		notify_service.PullRequestSynchronized(ctx, pusher, pr)

		// this always seems to be false
		isForcePush := comment != nil && comment.IsForcePush

		results = append(results, private.HookProcReceiveRefResult{
			OldOID:      oldCommitID,
			NewOID:      opts.NewCommitIDs[i],
			Ref:         pr.GetGitRefName(),
			OriginalRef: opts.RefFullNames[i],
			IsForcePush: isForcePush,
		})
	}

	return results, nil
}

// UserNameChanged handle user name change for agit flow pull
func UserNameChanged(ctx context.Context, user *user_model.User, newName string) error {
	pulls, err := issues_model.GetAllUnmergedAgitPullRequestByPoster(ctx, user.ID)
	if err != nil {
		return err
	}

	newName = strings.ToLower(newName)

	for _, pull := range pulls {
		pull.HeadBranch = strings.TrimPrefix(pull.HeadBranch, user.LowerName+"/")
		pull.HeadBranch = newName + "/" + pull.HeadBranch
		if err = pull.UpdateCols(ctx, "head_branch"); err != nil {
			return err
		}
	}

	return nil
}
