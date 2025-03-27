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

type comments struct {
	container
}

func (o *comments) ListPage(ctx context.Context, page int) generic.ChildrenSlice {
	pageSize := o.getPageSize()

	project := f3_tree.GetProjectID(o.GetNode())
	commentable := f3_tree.GetCommentableID(o.GetNode())

	issue, err := issues_model.GetIssueByIndex(ctx, project, commentable)
	if err != nil {
		panic(fmt.Errorf("GetIssueByIndex %v %w", commentable, err))
	}

	sess := db.GetEngine(ctx).
		Table("comment").
		Where("`issue_id` = ? AND `type` = ?", issue.ID, issues_model.CommentTypeComment)
	if page != 0 {
		sess = db.SetSessionPagination(sess, &db.ListOptions{Page: page, PageSize: pageSize})
	}
	forgejoComments := make([]*issues_model.Comment, 0, pageSize)
	if err := sess.Find(&forgejoComments); err != nil {
		panic(fmt.Errorf("error while listing comments: %v", err))
	}

	return f3_tree.ConvertListed(ctx, o.GetNode(), f3_tree.ConvertToAny(forgejoComments...)...)
}

func newComments() generic.NodeDriverInterface {
	return &comments{}
}
