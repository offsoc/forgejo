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

type milestones struct {
	container
}

func (o *milestones) ListPage(ctx context.Context, page int) generic.ChildrenSlice {
	pageSize := o.getPageSize()

	project := f3_tree.GetProjectID(o.GetNode())

	forgejoMilestones, err := db.Find[issues_model.Milestone](ctx, issues_model.FindMilestoneOptions{
		ListOptions: db.ListOptions{Page: page, PageSize: pageSize},
		RepoID:      project,
	})
	if err != nil {
		panic(fmt.Errorf("error while listing milestones: %v", err))
	}

	return f3_tree.ConvertListed(ctx, o.GetNode(), f3_tree.ConvertToAny(forgejoMilestones...)...)
}

func newMilestones() generic.NodeDriverInterface {
	return &milestones{}
}
