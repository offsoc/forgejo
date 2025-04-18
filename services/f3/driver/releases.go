// Copyright Earl Warren <contact@earl-warren.org>
// Copyright Loïc Dachary <loic@dachary.org>
// SPDX-License-Identifier: MIT

package driver

import (
	"context"
	"fmt"

	"forgejo.org/models/db"
	repo_model "forgejo.org/models/repo"

	f3_tree "code.forgejo.org/f3/gof3/v3/tree/f3"
	"code.forgejo.org/f3/gof3/v3/tree/generic"
)

type releases struct {
	container
}

func (o *releases) ListPage(ctx context.Context, page int) generic.ChildrenSlice {
	pageSize := o.getPageSize()

	project := f3_tree.GetProjectID(o.GetNode())

	forgejoReleases, err := db.Find[repo_model.Release](ctx, repo_model.FindReleasesOptions{
		ListOptions:   db.ListOptions{Page: page, PageSize: pageSize},
		IncludeDrafts: true,
		IncludeTags:   false,
		RepoID:        project,
	})
	if err != nil {
		panic(fmt.Errorf("error while listing releases: %v", err))
	}

	return f3_tree.ConvertListed(ctx, o.GetNode(), f3_tree.ConvertToAny(forgejoReleases...)...)
}

func newReleases() generic.NodeDriverInterface {
	return &releases{}
}
