// Copyright Earl Warren <contact@earl-warren.org>
// Copyright Loïc Dachary <loic@dachary.org>
// SPDX-License-Identifier: MIT

package driver

import (
	"context"

	"code.forgejo.org/f3/gof3/v3/f3"
	f3_id "code.forgejo.org/f3/gof3/v3/id"
	"code.forgejo.org/f3/gof3/v3/tree/generic"
)

type root struct {
	generic.NullDriver

	content f3.Interface
}

func newRoot(content f3.Interface) generic.NodeDriverInterface {
	return &root{
		content: content,
	}
}

func (o *root) FromFormat(content f3.Interface) {
	o.content = content
}

func (o *root) ToFormat() f3.Interface {
	return o.content
}

func (o *root) Get(context.Context) bool { return true }

func (o *root) Put(context.Context) f3_id.NodeID {
	return f3_id.NilID
}

func (o *root) Patch(context.Context) {
}
