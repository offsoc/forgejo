// Copyright Earl Warren <contact@earl-warren.org>
// Copyright Loïc Dachary <loic@dachary.org>
// SPDX-License-Identifier: MIT

package driver

import (
	"context"
	"fmt"

	"forgejo.org/models/db"
	user_model "forgejo.org/models/user"

	f3_id "code.forgejo.org/f3/gof3/v3/id"
	f3_tree "code.forgejo.org/f3/gof3/v3/tree/f3"
	"code.forgejo.org/f3/gof3/v3/tree/generic"
)

type users struct {
	container
}

func (o *users) ListPage(ctx context.Context, page int) generic.ChildrenSlice {
	sess := db.GetEngine(ctx).In("type", user_model.UserTypeIndividual, user_model.UserTypeRemoteUser)
	if page != 0 {
		sess = db.SetSessionPagination(sess, &db.ListOptions{Page: page, PageSize: o.getPageSize()})
	}
	sess = sess.Select("`user`.*")
	users := make([]*user_model.User, 0, o.getPageSize())

	if err := sess.Find(&users); err != nil {
		panic(fmt.Errorf("error while listing users: %v", err))
	}

	return f3_tree.ConvertListed(ctx, o.GetNode(), f3_tree.ConvertToAny(users...)...)
}

func (o *users) GetIDFromName(ctx context.Context, name string) f3_id.NodeID {
	user, err := user_model.GetUserByName(ctx, name)
	if err != nil {
		panic(fmt.Errorf("GetUserByName: %v", err))
	}

	return f3_id.NewNodeID(user.ID)
}

func newUsers() generic.NodeDriverInterface {
	return &users{}
}
