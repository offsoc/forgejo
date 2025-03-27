// Copyright 2023 The Forgejo Authors c/o Codeberg e.V.. All rights reserved.
// SPDX-License-Identifier: MIT

package doctor

import (
	"context"
	"strings"

	"forgejo.org/models/db"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/modules/log"

	"xorm.io/builder"
)

func FixPushMirrorsWithoutGitRemote(ctx context.Context, logger log.Logger, autofix bool) error {
	var missingMirrors []*repo_model.PushMirror

	err := db.Iterate(ctx, builder.Gt{"id": 0}, func(ctx context.Context, repo *repo_model.Repository) error {
		pushMirrors, _, err := repo_model.GetPushMirrorsByRepoID(ctx, repo.ID, db.ListOptions{})
		if err != nil {
			return err
		}

		for i := 0; i < len(pushMirrors); i++ {
			_, err = repo_model.GetPushMirrorRemoteAddress(repo.OwnerName, repo.Name, pushMirrors[i].RemoteName)
			if err != nil {
				if strings.Contains(err.Error(), "No such remote") {
					missingMirrors = append(missingMirrors, pushMirrors[i])
				} else if logger != nil {
					logger.Warn("Unable to retrieve the remote address of a mirror: %s", err)
				}
			}
		}

		return nil
	})
	if err != nil {
		if logger != nil {
			logger.Critical("Unable to iterate across repounits to fix push mirrors without a git remote: Error %v", err)
		}
		return err
	}

	count := len(missingMirrors)
	if !autofix {
		if logger != nil {
			if count == 0 {
				logger.Info("Found no push mirrors with missing git remotes")
			} else {
				logger.Warn("Found %d push mirrors with missing git remotes", count)
			}
		}
		return nil
	}

	for i := 0; i < len(missingMirrors); i++ {
		if logger != nil {
			logger.Info("Removing push mirror #%d (remote: %s), for repo: %s/%s",
				missingMirrors[i].ID,
				missingMirrors[i].RemoteName,
				missingMirrors[i].GetRepository(ctx).OwnerName,
				missingMirrors[i].GetRepository(ctx).Name)
		}

		err = repo_model.DeletePushMirrors(ctx, repo_model.PushMirrorOptions{
			ID:         missingMirrors[i].ID,
			RepoID:     missingMirrors[i].RepoID,
			RemoteName: missingMirrors[i].RemoteName,
		})
		if err != nil {
			if logger != nil {
				logger.Critical("Error removing a push mirror (repo_id: %d, push_mirror: %d): %s", missingMirrors[i].Repo.ID, missingMirrors[i].ID, err)
			}
			return err
		}
	}

	return nil
}

func init() {
	Register(&Check{
		Title:     "Check for push mirrors without a git remote configured",
		Name:      "fix-push-mirrors-without-git-remote",
		IsDefault: false,
		Run:       FixPushMirrorsWithoutGitRemote,
		Priority:  7,
	})
}
