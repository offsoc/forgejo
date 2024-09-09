// Copyright Earl Warren <contact@earl-warren.org>
// Copyright Loïc Dachary <loic@dachary.org>
// SPDX-License-Identifier: MIT

package driver

import (
	"context"
	"fmt"

	user_model "code.gitea.io/gitea/models/user"

	"code.forgejo.org/f3/gof3/v3/f3"
	f3_tree "code.forgejo.org/f3/gof3/v3/tree/f3"
	"code.forgejo.org/f3/gof3/v3/tree/generic"
	"code.forgejo.org/f3/gof3/v3/util"
)

type forge struct {
	generic.NullDriver

	ownersKind map[string]generic.Kind
}

func newForge() generic.NodeDriverInterface {
	return &forge{
		ownersKind: make(map[string]generic.Kind),
	}
}

func (o *forge) getOwnersKind(ctx context.Context, id string) generic.Kind {
	kind, ok := o.ownersKind[id]
	if !ok {
		user, err := user_model.GetUserByID(ctx, util.ParseInt(id))
		if err != nil {
			panic(fmt.Errorf("user_repo.GetUserByID: %w", err))
		}
		kind = f3_tree.KindUsers
		if user.IsOrganization() {
			kind = f3_tree.KindOrganization
		}
		o.ownersKind[id] = kind
	}
	return kind
}

func (o *forge) getOwnersPath(ctx context.Context, id string) f3_tree.Path {
	return f3_tree.NewPathFromString("/").SetForge().SetOwners(o.getOwnersKind(ctx, id))
}

func (o *forge) Equals(context.Context, generic.NodeInterface) bool { return true }
func (o *forge) Get(context.Context) bool                           { return true }
func (o *forge) Put(context.Context) generic.NodeID                 { return generic.NewNodeID("forge") }
func (o *forge) Patch(context.Context)                              {}
func (o *forge) Delete(context.Context)                             {}
func (o *forge) NewFormat() f3.Interface                            { return &f3.Forge{} }
func (o *forge) FromFormat(f3.Interface)                            {}

func (o *forge) ToFormat() f3.Interface {
	return &f3.Forge{
		Common: f3.NewCommon("forge"),
		URL:    o.String(),
	}
}
