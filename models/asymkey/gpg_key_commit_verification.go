// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package asymkey

import (
	"context"

	repo_model "forgejo.org/models/repo"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/git"
)

//   __________________  ________   ____  __.
//  /  _____/\______   \/  _____/  |    |/ _|____ ___.__.
// /   \  ___ |     ___/   \  ___  |      <_/ __ <   |  |
// \    \_\  \|    |   \    \_\  \ |    |  \  ___/\___  |
//  \______  /|____|    \______  / |____|__ \___  > ____|
//         \/                  \/          \/   \/\/
// _________                        .__  __
// \_   ___ \  ____   _____   _____ |__|/  |_
// /    \  \/ /  _ \ /     \ /     \|  \   __\
// \     \___(  <_> )  Y Y  \  Y Y  \  ||  |
//  \______  /\____/|__|_|  /__|_|  /__||__|
//         \/             \/      \/
// ____   ____           .__  _____.__               __  .__
// \   \ /   /___________|__|/ ____\__| ____ _____ _/  |_|__| ____   ____
//  \   Y   // __ \_  __ \  \   __\|  |/ ___\\__  \\   __\  |/  _ \ /    \
//   \     /\  ___/|  | \/  ||  |  |  \  \___ / __ \|  | |  (  <_> )   |  \
//    \___/  \___  >__|  |__||__|  |__|\___  >____  /__| |__|\____/|___|  /
//               \/                        \/     \/                    \/

// This file provides functions relating commit verification

// SignCommit represents a commit with validation of signature.
type SignCommit struct {
	Verification *ObjectVerification
	*user_model.UserCommit
}

// ParseCommitsWithSignature checks if signaute of commits are corresponding to users gpg keys.
func ParseCommitsWithSignature(ctx context.Context, oldCommits []*user_model.UserCommit, repoTrustModel repo_model.TrustModelType, isOwnerMemberCollaborator func(*user_model.User) (bool, error)) []*SignCommit {
	newCommits := make([]*SignCommit, 0, len(oldCommits))
	keyMap := map[string]bool{}

	for _, c := range oldCommits {
		o := commitToGitObject(c.Commit)
		signCommit := &SignCommit{
			UserCommit:   c,
			Verification: ParseObjectWithSignature(ctx, &o),
		}

		_ = CalculateTrustStatus(signCommit.Verification, repoTrustModel, isOwnerMemberCollaborator, &keyMap)

		newCommits = append(newCommits, signCommit)
	}
	return newCommits
}

func ParseCommitWithSignature(ctx context.Context, c *git.Commit) *ObjectVerification {
	o := commitToGitObject(c)
	return ParseObjectWithSignature(ctx, &o)
}
