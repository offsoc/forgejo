// Copyright Earl Warren <contact@earl-warren.org>
// Copyright Loïc Dachary <loic@dachary.org>
// SPDX-License-Identifier: MIT

package driver

import (
	"context"
	"fmt"

	"forgejo.org/models/db"
	repo_model "forgejo.org/models/repo"

	"code.forgejo.org/f3/gof3/v3/f3"
	f3_id "code.forgejo.org/f3/gof3/v3/id"
	f3_tree "code.forgejo.org/f3/gof3/v3/tree/f3"
	"code.forgejo.org/f3/gof3/v3/tree/generic"
	f3_util "code.forgejo.org/f3/gof3/v3/util"
)

var _ f3_tree.ForgeDriverInterface = &topic{}

type topic struct {
	common

	forgejoTopic *repo_model.Topic
}

func (o *topic) SetNative(topic any) {
	o.forgejoTopic = topic.(*repo_model.Topic)
}

func (o *topic) GetNativeID() string {
	return fmt.Sprintf("%d", o.forgejoTopic.ID)
}

func (o *topic) NewFormat() f3.Interface {
	node := o.GetNode()
	return node.GetTree().(f3_tree.TreeInterface).NewFormat(node.GetKind())
}

func (o *topic) ToFormat() f3.Interface {
	if o.forgejoTopic == nil {
		return o.NewFormat()
	}

	return &f3.Topic{
		Common: f3.NewCommon(o.GetNativeID()),
		Name:   o.forgejoTopic.Name,
	}
}

func (o *topic) FromFormat(content f3.Interface) {
	topic := content.(*f3.Topic)
	o.forgejoTopic = &repo_model.Topic{
		ID:   f3_util.ParseInt(topic.GetID()),
		Name: topic.Name,
	}
}

func (o *topic) Get(ctx context.Context) bool {
	node := o.GetNode()
	o.Trace("%s", node.GetID())

	id := node.GetID().Int64()

	if has, err := db.GetEngine(ctx).Where("ID = ?", id).Get(o.forgejoTopic); err != nil {
		panic(fmt.Errorf("topic %v %w", id, err))
	} else if !has {
		return false
	}

	return true
}

func (o *topic) Patch(ctx context.Context) {
	o.Trace("%d", o.forgejoTopic.ID)
	if _, err := db.GetEngine(ctx).ID(o.forgejoTopic.ID).Cols("name").Update(o.forgejoTopic); err != nil {
		panic(fmt.Errorf("UpdateTopicCols: %v %v", o.forgejoTopic, err))
	}
}

func (o *topic) Put(ctx context.Context) f3_id.NodeID {
	sess := db.GetEngine(ctx)

	if _, err := sess.Insert(o.forgejoTopic); err != nil {
		panic(err)
	}
	o.Trace("topic created %d", o.forgejoTopic.ID)
	return f3_id.NewNodeID(o.forgejoTopic.ID)
}

func (o *topic) Delete(ctx context.Context) {
	node := o.GetNode()
	o.Trace("%s", node.GetID())

	sess := db.GetEngine(ctx)

	if _, err := sess.Delete(&repo_model.RepoTopic{
		TopicID: o.forgejoTopic.ID,
	}); err != nil {
		panic(fmt.Errorf("Delete RepoTopic for %v %v", o.forgejoTopic, err))
	}

	if _, err := sess.Delete(o.forgejoTopic); err != nil {
		panic(fmt.Errorf("Delete Topic %v %v", o.forgejoTopic, err))
	}
}

func newTopic() generic.NodeDriverInterface {
	return &topic{}
}
