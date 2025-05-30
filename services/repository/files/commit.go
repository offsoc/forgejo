// Copyright 2019 The Gitea Authors. All rights reserved.
// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package files

import (
	"context"

	asymkey_model "forgejo.org/models/asymkey"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/modules/git"
	"forgejo.org/modules/structs"
)

// CountDivergingCommits determines how many commits a branch is ahead or behind the repository's base branch
func CountDivergingCommits(ctx context.Context, repo *repo_model.Repository, branch string) (*git.DivergeObject, error) {
	divergence, err := git.GetDivergingCommits(ctx, repo.RepoPath(), repo.DefaultBranch, branch, nil)
	if err != nil {
		return nil, err
	}
	return &divergence, nil
}

// GetPayloadCommitVerification returns the verification information of a commit
func GetPayloadCommitVerification(ctx context.Context, commit *git.Commit) *structs.PayloadCommitVerification {
	verification := &structs.PayloadCommitVerification{}
	commitVerification := asymkey_model.ParseCommitWithSignature(ctx, commit)
	if commit.Signature != nil {
		verification.Signature = commit.Signature.Signature
		verification.Payload = commit.Signature.Payload
	}
	if commitVerification.SigningUser != nil {
		verification.Signer = &structs.PayloadUser{
			Name:  commitVerification.SigningUser.Name,
			Email: commitVerification.SigningUser.Email,
		}
	}
	verification.Verified = commitVerification.Verified
	verification.Reason = commitVerification.Reason
	if verification.Reason == "" && !verification.Verified {
		verification.Reason = asymkey_model.NotSigned
	}
	return verification
}
