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

type reviews struct {
	container
}

func (o *reviews) ListPage(ctx context.Context, page int) generic.ChildrenSlice {
	pageSize := o.getPageSize()

	project := f3_tree.GetProjectID(o.GetNode())
	pullRequest := f3_tree.GetPullRequestID(o.GetNode())

	issue, err := issues_model.GetIssueByIndex(ctx, project, pullRequest)
	if err != nil {
		panic(fmt.Errorf("GetIssueByIndex %v %w", pullRequest, err))
	}

	sess := db.GetEngine(ctx).
		Table("review").
		Where("`issue_id` = ?", issue.ID)
	if page != 0 {
		sess = db.SetSessionPagination(sess, &db.ListOptions{Page: page, PageSize: pageSize})
	}
	forgejoReviews := make([]*issues_model.Review, 0, pageSize)
	if err := sess.Find(&forgejoReviews); err != nil {
		panic(fmt.Errorf("error while listing reviews: %v", err))
	}

	return f3_tree.ConvertListed(ctx, o.GetNode(), f3_tree.ConvertToAny(forgejoReviews...)...)
}

func newReviews() generic.NodeDriverInterface {
	return &reviews{}
}
