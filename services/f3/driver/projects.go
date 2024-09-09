// Copyright Earl Warren <contact@earl-warren.org>
// Copyright Loïc Dachary <loic@dachary.org>
// SPDX-License-Identifier: MIT

package driver

import (
	"context"
	"fmt"

	"code.gitea.io/gitea/models/db"
	repo_model "code.gitea.io/gitea/models/repo"

	f3_tree "code.forgejo.org/f3/gof3/v3/tree/f3"
	"code.forgejo.org/f3/gof3/v3/tree/generic"
)

type projects struct {
	container
}

func (o *projects) GetIDFromName(ctx context.Context, name string) generic.NodeID {
	owner := f3_tree.GetOwnerName(o.GetNode())
	forgejoProject, err := repo_model.GetRepositoryByOwnerAndName(ctx, owner, name)
	if repo_model.IsErrRepoNotExist(err) {
		return generic.NilID
	}

	if err != nil {
		panic(fmt.Errorf("error GetRepositoryByOwnerAndName(%s, %s): %v", owner, name, err))
	}

	return generic.NewNodeID(forgejoProject.ID)
}

func (o *projects) ListPage(ctx context.Context, page int) generic.ChildrenSlice {
	pageSize := o.getPageSize()

	owner := f3_tree.GetOwner(o.GetNode())

	forgejoProjects, _, err := repo_model.SearchRepository(ctx, &repo_model.SearchRepoOptions{
		ListOptions: db.ListOptions{Page: page, PageSize: pageSize},
		OwnerID:     owner.GetID().Int64(),
		Private:     true,
	})
	if err != nil {
		panic(fmt.Errorf("error while listing projects: %v", err))
	}

	return f3_tree.ConvertListed(ctx, o.GetNode(), f3_tree.ConvertToAny(forgejoProjects...)...)
}

func newProjects() generic.NodeDriverInterface {
	return &projects{}
}
