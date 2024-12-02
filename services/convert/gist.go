// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package convert

import (
	"context"

	gist_model "code.gitea.io/gitea/models/gist"
	user_model "code.gitea.io/gitea/models/user"
	api "code.gitea.io/gitea/modules/structs"
)

// ToGistList convert a gist_model.Gist to an api.Gist
func ToGist(ctx context.Context, gist *gist_model.Gist, doer *user_model.User) *api.Gist {
	result := &api.Gist{
		ID:          gist.ID,
		UUID:        gist.UUID,
		Name:        gist.Name,
		Description: gist.Description,
		Visibility:  gist.Visibility.String(),
		Created:     gist.CreatedUnix.AsTime(),
		Updated:     gist.UpdatedUnix.AsTime(),
	}

	if gist.Owner != nil {
		result.Owner = ToUser(ctx, gist.Owner, doer)
	}

	return result
}

// ToGistList convert a gist_model.GistList to an api.GistList
func ToGistList(ctx context.Context, gistList gist_model.GistList, doer *user_model.User) *api.GistList {
	newList := make([]*api.Gist, len(gistList))

	for pos, gist := range gistList {
		newList[pos] = ToGist(ctx, gist, doer)
	}

	return &api.GistList{Gists: newList}
}
