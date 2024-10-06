// Copyright Earl Warren <contact@earl-warren.org>
// Copyright Loïc Dachary <loic@dachary.org>
// SPDX-License-Identifier: MIT

package driver

import (
	"context"

	"code.forgejo.org/f3/gof3/v3/f3"
	f3_tree "code.forgejo.org/f3/gof3/v3/tree/f3"
	"code.forgejo.org/f3/gof3/v3/tree/generic"
)

type container struct {
	common
}

func (o *container) NewFormat() f3.Interface {
	node := o.GetNode()
	return node.GetTree().(f3_tree.TreeInterface).NewFormat(node.GetKind())
}

func (o *container) ToFormat() f3.Interface {
	return o.NewFormat()
}

func (o *container) FromFormat(content f3.Interface) {
}

func (o *container) Get(context.Context) bool { return true }

func (o *container) Put(ctx context.Context) generic.NodeID {
	return o.upsert(ctx)
}

func (o *container) Patch(ctx context.Context) {
	o.upsert(ctx)
}

func (o *container) upsert(context.Context) generic.NodeID {
	return generic.NewNodeID(o.getKind())
}
