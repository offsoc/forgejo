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

type topics struct {
	container
}

func (o *topics) ListPage(ctx context.Context, page int) generic.ChildrenSlice {
	pageSize := o.getPageSize()

	sess := db.GetEngine(ctx)
	if page != 0 {
		sess = db.SetSessionPagination(sess, &db.ListOptions{Page: page, PageSize: pageSize})
	}
	sess = sess.Select("`topic`.*")
	topics := make([]*repo_model.Topic, 0, pageSize)

	if err := sess.Find(&topics); err != nil {
		panic(fmt.Errorf("error while listing topics: %v", err))
	}

	return f3_tree.ConvertListed(ctx, o.GetNode(), f3_tree.ConvertToAny(topics...)...)
}

func newTopics() generic.NodeDriverInterface {
	return &topics{}
}
