// Copyright Earl Warren <contact@earl-warren.org>
// Copyright Loïc Dachary <loic@dachary.org>
// SPDX-License-Identifier: MIT

package driver

import (
	"context"
	"fmt"
	"strings"

	"forgejo.org/models/db"
	issues_model "forgejo.org/models/issues"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/timeutil"

	"code.forgejo.org/f3/gof3/v3/f3"
	f3_id "code.forgejo.org/f3/gof3/v3/id"
	f3_tree "code.forgejo.org/f3/gof3/v3/tree/f3"
	"code.forgejo.org/f3/gof3/v3/tree/generic"
	f3_util "code.forgejo.org/f3/gof3/v3/util"
)

var _ f3_tree.ForgeDriverInterface = &reviewComment{}

type reviewComment struct {
	common

	forgejoReviewComment *issues_model.Comment
}

func (o *reviewComment) SetNative(reviewComment any) {
	o.forgejoReviewComment = reviewComment.(*issues_model.Comment)
}

func (o *reviewComment) GetNativeID() string {
	return fmt.Sprintf("%d", o.forgejoReviewComment.ID)
}

func (o *reviewComment) NewFormat() f3.Interface {
	node := o.GetNode()
	return node.GetTree().(f3_tree.TreeInterface).NewFormat(node.GetKind())
}

func patch2diff(patch string) string {
	split := strings.Split(patch, "\n@@")
	if len(split) == 2 {
		return "@@" + split[1]
	}
	return patch
}

func (o *reviewComment) ToFormat() f3.Interface {
	if o.forgejoReviewComment == nil {
		return o.NewFormat()
	}

	return &f3.ReviewComment{
		Common:    f3.NewCommon(o.GetNativeID()),
		PosterID:  f3_tree.NewUserReference(o.forgejoReviewComment.Poster.ID),
		Content:   o.forgejoReviewComment.Content,
		TreePath:  o.forgejoReviewComment.TreePath,
		DiffHunk:  patch2diff(o.forgejoReviewComment.PatchQuoted),
		Line:      int(o.forgejoReviewComment.Line),
		CommitID:  o.forgejoReviewComment.CommitSHA,
		CreatedAt: o.forgejoReviewComment.CreatedUnix.AsTime(),
		UpdatedAt: o.forgejoReviewComment.UpdatedUnix.AsTime(),
	}
}

func (o *reviewComment) FromFormat(content f3.Interface) {
	reviewComment := content.(*f3.ReviewComment)
	o.forgejoReviewComment = &issues_model.Comment{
		ID:       f3_util.ParseInt(reviewComment.GetID()),
		PosterID: reviewComment.PosterID.GetIDAsInt(),
		Poster: &user_model.User{
			ID: reviewComment.PosterID.GetIDAsInt(),
		},
		TreePath: reviewComment.TreePath,
		Content:  reviewComment.Content,
		// a hunk misses the patch header but it is never used so do not bother
		// reconstructing it
		Patch:       reviewComment.DiffHunk,
		PatchQuoted: reviewComment.DiffHunk,
		Line:        int64(reviewComment.Line),
		CommitSHA:   reviewComment.CommitID,
		CreatedUnix: timeutil.TimeStamp(reviewComment.CreatedAt.Unix()),
		UpdatedUnix: timeutil.TimeStamp(reviewComment.UpdatedAt.Unix()),
	}
}

func (o *reviewComment) Get(ctx context.Context) bool {
	node := o.GetNode()
	o.Trace("%s", node.GetID())

	id := node.GetID().Int64()

	reviewComment, err := issues_model.GetCommentByID(ctx, id)
	if issues_model.IsErrCommentNotExist(err) {
		return false
	}
	if err != nil {
		panic(fmt.Errorf("reviewComment %v %w", id, err))
	}
	if err := reviewComment.LoadPoster(ctx); err != nil {
		panic(fmt.Errorf("LoadPoster %v %w", *reviewComment, err))
	}
	o.forgejoReviewComment = reviewComment
	return true
}

func (o *reviewComment) Patch(ctx context.Context) {
	o.Trace("%d", o.forgejoReviewComment.ID)
	if _, err := db.GetEngine(ctx).ID(o.forgejoReviewComment.ID).Cols("content").Update(o.forgejoReviewComment); err != nil {
		panic(fmt.Errorf("UpdateReviewCommentCols: %v %v", o.forgejoReviewComment, err))
	}
}

func (o *reviewComment) Put(ctx context.Context) f3_id.NodeID {
	node := o.GetNode()
	o.Trace("%s", node.GetID())

	sess := db.GetEngine(ctx)

	if _, err := sess.NoAutoTime().Insert(o.forgejoReviewComment); err != nil {
		panic(err)
	}
	o.Trace("reviewComment created %d", o.forgejoReviewComment.ID)
	return f3_id.NewNodeID(o.forgejoReviewComment.ID)
}

func (o *reviewComment) Delete(ctx context.Context) {
	node := o.GetNode()
	o.Trace("%s", node.GetID())

	if err := issues_model.DeleteComment(ctx, o.forgejoReviewComment); err != nil {
		panic(err)
	}
}

func newReviewComment() generic.NodeDriverInterface {
	return &reviewComment{}
}
