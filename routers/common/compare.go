// Copyright 2024 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package common

import (
	repo_model "forgejo.org/models/repo"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/git"
)

// CompareInfo represents the collected results from ParseCompareInfo
type CompareInfo struct {
	HeadUser         *user_model.User
	HeadRepo         *repo_model.Repository
	HeadGitRepo      *git.Repository
	CompareInfo      *git.CompareInfo
	BaseBranch       string
	HeadBranch       string
	DirectComparison bool
}
