// Copyright 2019 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package integrations

import (
	"code.gitea.io/gitea/models"
	api "code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/services/repository/repofiles"
)

func createFileInBranch(user *models.User, repo *models.Repository, treePath, branchName, content string) (*api.FileResponse, error) {
	opts := &repofiles.UpdateRepoFileOptions{
		OldBranch: branchName,
		TreePath:  treePath,
		Content:   content,
		IsNewFile: true,
		Author:    nil,
		Committer: nil,
	}
	return repofiles.CreateOrUpdateRepoFile(repo, user, opts)
}

func createFile(user *models.User, repo *models.Repository, treePath string) (*api.FileResponse, error) {
	return createFileInBranch(user, repo, treePath, repo.DefaultBranch, "This is a NEW file")
}
