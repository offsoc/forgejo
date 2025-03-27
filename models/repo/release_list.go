// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package repo

import (
	"context"

	user_model "forgejo.org/models/user"
)

type ReleaseList []*Release

// LoadAttributes loads the repository and publisher for the releases.
func (r ReleaseList) LoadAttributes(ctx context.Context) error {
	repoCache := make(map[int64]*Repository)
	userCache := make(map[int64]*user_model.User)

	for _, release := range r {
		var err error
		repo, ok := repoCache[release.RepoID]
		if !ok {
			repo, err = GetRepositoryByID(ctx, release.RepoID)
			if err != nil {
				return err
			}
			repoCache[release.RepoID] = repo
		}
		release.Repo = repo

		publisher, ok := userCache[release.PublisherID]
		if !ok {
			publisher, err = user_model.GetUserByID(ctx, release.PublisherID)
			if err != nil {
				if !user_model.IsErrUserNotExist(err) {
					return err
				}
				publisher = user_model.NewGhostUser()
			}
			userCache[release.PublisherID] = publisher
		}
		release.Publisher = publisher
	}
	return nil
}
