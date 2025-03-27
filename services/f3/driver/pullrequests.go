// Copyright Earl Warren <contact@earl-warren.org>
// Copyright Loïc Dachary <loic@dachary.org>
// SPDX-License-Identifier: MIT

package driver

import (
	"context"
	"fmt"

	"forgejo.org/models/db"
	issues_model "forgejo.org/models/issues"
	"forgejo.org/modules/optional"

	f3_tree "code.forgejo.org/f3/gof3/v3/tree/f3"
	"code.forgejo.org/f3/gof3/v3/tree/generic"
)

type pullRequests struct {
	container
}

func (o *pullRequests) ListPage(ctx context.Context, page int) generic.ChildrenSlice {
	pageSize := o.getPageSize()

	project := f3_tree.GetProjectID(o.GetNode())

	forgejoPullRequests, err := issues_model.Issues(ctx, &issues_model.IssuesOptions{
		Paginator: &db.ListOptions{Page: page, PageSize: pageSize},
		RepoIDs:   []int64{project},
		IsPull:    optional.Some(true),
	})
	if err != nil {
		panic(fmt.Errorf("error while listing pullRequests: %v", err))
	}

	return f3_tree.ConvertListed(ctx, o.GetNode(), f3_tree.ConvertToAny(forgejoPullRequests...)...)
}

func newPullRequests() generic.NodeDriverInterface {
	return &pullRequests{}
}
