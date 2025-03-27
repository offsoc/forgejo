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
	"xorm.io/builder"
)

type reactions struct {
	container
}

func (o *reactions) ListPage(ctx context.Context, page int) generic.ChildrenSlice {
	pageSize := o.getPageSize()

	reactionable := f3_tree.GetReactionable(o.GetNode())
	reactionableID := f3_tree.GetReactionableID(o.GetNode())

	sess := db.GetEngine(ctx)
	cond := builder.NewCond()
	switch reactionable.GetKind() {
	case f3_tree.KindIssue, f3_tree.KindPullRequest:
		project := f3_tree.GetProjectID(o.GetNode())
		issue, err := issues_model.GetIssueByIndex(ctx, project, reactionableID)
		if err != nil {
			panic(fmt.Errorf("GetIssueByIndex %v %w", reactionableID, err))
		}
		cond = cond.And(builder.Eq{"reaction.issue_id": issue.ID})
	case f3_tree.KindComment:
		cond = cond.And(builder.Eq{"reaction.comment_id": reactionableID})
	default:
		panic(fmt.Errorf("unexpected type %v", reactionable.GetKind()))
	}

	sess = sess.Where(cond)
	if page > 0 {
		sess = db.SetSessionPagination(sess, &db.ListOptions{Page: page, PageSize: pageSize})
	}
	reactions := make([]*issues_model.Reaction, 0, 10)
	if err := sess.Find(&reactions); err != nil {
		panic(fmt.Errorf("error while listing reactions: %v", err))
	}

	return f3_tree.ConvertListed(ctx, o.GetNode(), f3_tree.ConvertToAny(reactions...)...)
}

func newReactions() generic.NodeDriverInterface {
	return &reactions{}
}
