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

type reviewComments struct {
	container
}

func (o *reviewComments) ListPage(ctx context.Context, page int) generic.ChildrenSlice {
	pageSize := o.getPageSize()

	id := f3_tree.GetReviewID(o.GetNode())

	sess := db.GetEngine(ctx).
		Table("comment").
		Where("`review_id` = ? AND `type` = ?", id, issues_model.CommentTypeCode)
	if page != 0 {
		sess = db.SetSessionPagination(sess, &db.ListOptions{Page: page, PageSize: pageSize})
	}
	forgejoReviewComments := make([]*issues_model.Comment, 0, pageSize)
	if err := sess.Find(&forgejoReviewComments); err != nil {
		panic(fmt.Errorf("error while listing reviewComments: %v", err))
	}

	return f3_tree.ConvertListed(ctx, o.GetNode(), f3_tree.ConvertToAny(forgejoReviewComments...)...)
}

func newReviewComments() generic.NodeDriverInterface {
	return &reviewComments{}
}
