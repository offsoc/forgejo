// Copyright Earl Warren <contact@earl-warren.org>
// Copyright Loïc Dachary <loic@dachary.org>
// SPDX-License-Identifier: MIT

package driver

import (
	"context"
	"fmt"

	repo_model "forgejo.org/models/repo"

	f3_tree "code.forgejo.org/f3/gof3/v3/tree/f3"
	"code.forgejo.org/f3/gof3/v3/tree/generic"
)

type assets struct {
	container
}

func (o *assets) ListPage(ctx context.Context, page int) generic.ChildrenSlice {
	if page > 1 {
		return generic.NewChildrenSlice(0)
	}

	releaseID := f3_tree.GetReleaseID(o.GetNode())

	release, err := repo_model.GetReleaseByID(ctx, releaseID)
	if err != nil {
		panic(fmt.Errorf("GetReleaseByID %v %w", releaseID, err))
	}

	if err := release.LoadAttributes(ctx); err != nil {
		panic(fmt.Errorf("error while listing assets: %v", err))
	}

	return f3_tree.ConvertListed(ctx, o.GetNode(), f3_tree.ConvertToAny(release.Attachments...)...)
}

func newAssets() generic.NodeDriverInterface {
	return &assets{}
}
