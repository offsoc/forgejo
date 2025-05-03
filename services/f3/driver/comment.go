// Copyright Earl Warren <contact@earl-warren.org>
// Copyright Loïc Dachary <loic@dachary.org>
// SPDX-License-Identifier: MIT

package driver

import (
	"context"
	"fmt"

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

var _ f3_tree.ForgeDriverInterface = &comment{}

type comment struct {
	common

	forgejoComment *issues_model.Comment
}

func (o *comment) SetNative(comment any) {
	o.forgejoComment = comment.(*issues_model.Comment)
}

func (o *comment) GetNativeID() string {
	return fmt.Sprintf("%d", o.forgejoComment.ID)
}

func (o *comment) NewFormat() f3.Interface {
	node := o.GetNode()
	return node.GetTree().(f3_tree.TreeInterface).NewFormat(node.GetKind())
}

func (o *comment) ToFormat() f3.Interface {
	if o.forgejoComment == nil {
		return o.NewFormat()
	}
	return &f3.Comment{
		Common:   f3.NewCommon(fmt.Sprintf("%d", o.forgejoComment.ID)),
		PosterID: f3_tree.NewUserReference(o.forgejoComment.Poster.ID),
		Content:  o.forgejoComment.Content,
		Created:  o.forgejoComment.CreatedUnix.AsTime(),
		Updated:  o.forgejoComment.UpdatedUnix.AsTime(),
	}
}

func (o *comment) FromFormat(content f3.Interface) {
	comment := content.(*f3.Comment)

	o.forgejoComment = &issues_model.Comment{
		ID:       f3_util.ParseInt(comment.GetID()),
		PosterID: comment.PosterID.GetIDAsInt(),
		Poster: &user_model.User{
			ID: comment.PosterID.GetIDAsInt(),
		},
		Content:     comment.Content,
		CreatedUnix: timeutil.TimeStamp(comment.Created.Unix()),
		UpdatedUnix: timeutil.TimeStamp(comment.Updated.Unix()),
	}
}

func (o *comment) Get(ctx context.Context) bool {
	node := o.GetNode()
	o.Trace("%s", node.GetID())

	id := node.GetID().Int64()

	comment, err := issues_model.GetCommentByID(ctx, id)
	if issues_model.IsErrCommentNotExist(err) {
		return false
	}
	if err != nil {
		panic(fmt.Errorf("comment %v %w", id, err))
	}
	if err := comment.LoadPoster(ctx); err != nil {
		panic(fmt.Errorf("LoadPoster %v %w", *comment, err))
	}
	o.forgejoComment = comment
	return true
}

func (o *comment) Patch(ctx context.Context) {
	o.Trace("%d", o.forgejoComment.ID)
	if _, err := db.GetEngine(ctx).ID(o.forgejoComment.ID).Cols("content").Update(o.forgejoComment); err != nil {
		panic(fmt.Errorf("UpdateCommentCols: %v %v", o.forgejoComment, err))
	}
}

func (o *comment) Put(ctx context.Context) f3_id.NodeID {
	node := o.GetNode()
	o.Trace("%s", node.GetID())

	sess := db.GetEngine(ctx)

	if _, err := sess.NoAutoTime().Insert(o.forgejoComment); err != nil {
		panic(err)
	}
	o.Trace("comment created %d", o.forgejoComment.ID)
	return f3_id.NewNodeID(o.forgejoComment.ID)
}

func (o *comment) Delete(ctx context.Context) {
	node := o.GetNode()
	o.Trace("%s", node.GetID())

	if err := issues_model.DeleteComment(ctx, o.forgejoComment); err != nil {
		panic(err)
	}
}

func newComment() generic.NodeDriverInterface {
	return &comment{}
}
