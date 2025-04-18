// Copyright Earl Warren <contact@earl-warren.org>
// Copyright Loïc Dachary <loic@dachary.org>
// SPDX-License-Identifier: MIT

package driver

import (
	"context"
	"fmt"

	"forgejo.org/models/db"
	issues_model "forgejo.org/models/issues"

	f3_tree "code.forgejo.org/f3/gof3/v3/tree/f3"
	"code.forgejo.org/f3/gof3/v3/tree/generic"
)

type issues struct {
	container
}

func (o *issues) ListPage(ctx context.Context, page int) generic.ChildrenSlice {
	pageSize := o.getPageSize()

	project := f3_tree.GetProjectID(o.GetNode())

	forgejoIssues, err := issues_model.Issues(ctx, &issues_model.IssuesOptions{
		Paginator: &db.ListOptions{Page: page, PageSize: pageSize},
		RepoIDs:   []int64{project},
	})
	if err != nil {
		panic(fmt.Errorf("error while listing issues: %v", err))
	}

	return f3_tree.ConvertListed(ctx, o.GetNode(), f3_tree.ConvertToAny(forgejoIssues...)...)
}

func newIssues() generic.NodeDriverInterface {
	return &issues{}
}
