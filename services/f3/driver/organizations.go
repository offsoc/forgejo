// Copyright Earl Warren <contact@earl-warren.org>
// Copyright Loïc Dachary <loic@dachary.org>
// SPDX-License-Identifier: MIT

package driver

import (
	"context"
	"fmt"

	"forgejo.org/models/db"
	org_model "forgejo.org/models/organization"
	user_model "forgejo.org/models/user"

	f3_id "code.forgejo.org/f3/gof3/v3/id"
	f3_tree "code.forgejo.org/f3/gof3/v3/tree/f3"
	"code.forgejo.org/f3/gof3/v3/tree/generic"
)

type organizations struct {
	container
}

func (o *organizations) ListPage(ctx context.Context, page int) generic.ChildrenSlice {
	sess := db.GetEngine(ctx)
	if page != 0 {
		sess = db.SetSessionPagination(sess, &db.ListOptions{Page: page, PageSize: o.getPageSize()})
	}
	sess = sess.Select("`user`.*").
		Where("`type`=?", user_model.UserTypeOrganization)
	organizations := make([]*org_model.Organization, 0, o.getPageSize())

	if err := sess.Find(&organizations); err != nil {
		panic(fmt.Errorf("error while listing organizations: %v", err))
	}

	return f3_tree.ConvertListed(ctx, o.GetNode(), f3_tree.ConvertToAny(organizations...)...)
}

func (o *organizations) GetIDFromName(ctx context.Context, name string) f3_id.NodeID {
	organization, err := org_model.GetOrgByName(ctx, name)
	if err != nil {
		panic(fmt.Errorf("GetOrganizationByName: %v", err))
	}

	return f3_id.NewNodeID(organization.ID)
}

func newOrganizations() generic.NodeDriverInterface {
	return &organizations{}
}
