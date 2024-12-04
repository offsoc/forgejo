// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package gist

import (
	"context"

	user_model "code.gitea.io/gitea/models/user"
)

type GistList []*Gist //revive:disable-line:exported

func (gistList GistList) LoadOwner(ctx context.Context) error {
	ownerCache := make(map[int64]*user_model.User)

	for _, gist := range gistList {
		gist.Owner = ownerCache[gist.OwnerID]
		if gist.Owner != nil {
			continue
		}

		err := gist.LoadOwner(ctx)
		if err != nil {
			return err
		}

		ownerCache[gist.OwnerID] = gist.Owner
	}

	return nil
}
