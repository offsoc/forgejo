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

	"code.forgejo.org/f3/gof3/v3/f3"
	f3_id "code.forgejo.org/f3/gof3/v3/id"
	f3_tree "code.forgejo.org/f3/gof3/v3/tree/f3"
	"code.forgejo.org/f3/gof3/v3/tree/generic"
	f3_util "code.forgejo.org/f3/gof3/v3/util"
)

var _ f3_tree.ForgeDriverInterface = &organization{}

type organization struct {
	common

	forgejoOrganization *org_model.Organization
}

func (o *organization) SetNative(organization any) {
	o.forgejoOrganization = organization.(*org_model.Organization)
}

func (o *organization) GetNativeID() string {
	return fmt.Sprintf("%d", o.forgejoOrganization.ID)
}

func (o *organization) NewFormat() f3.Interface {
	node := o.GetNode()
	return node.GetTree().(f3_tree.TreeInterface).NewFormat(node.GetKind())
}

func (o *organization) ToFormat() f3.Interface {
	if o.forgejoOrganization == nil {
		return o.NewFormat()
	}
	return &f3.Organization{
		Common:   f3.NewCommon(fmt.Sprintf("%d", o.forgejoOrganization.ID)),
		Name:     o.forgejoOrganization.Name,
		FullName: o.forgejoOrganization.FullName,
	}
}

func (o *organization) FromFormat(content f3.Interface) {
	organization := content.(*f3.Organization)
	o.forgejoOrganization = &org_model.Organization{
		ID:       f3_util.ParseInt(organization.GetID()),
		Name:     organization.Name,
		FullName: organization.FullName,
	}
}

func (o *organization) Get(ctx context.Context) bool {
	node := o.GetNode()
	o.Trace("%s", node.GetID())
	id := node.GetID().Int64()
	organization, err := org_model.GetOrgByID(ctx, id)
	if user_model.IsErrUserNotExist(err) {
		return false
	}
	if err != nil {
		panic(fmt.Errorf("organization %v %w", id, err))
	}
	o.forgejoOrganization = organization
	return true
}

func (o *organization) Patch(ctx context.Context) {
	o.Trace("%d", o.forgejoOrganization.ID)
	if _, err := db.GetEngine(ctx).ID(o.forgejoOrganization.ID).Cols("full_name").Update(o.forgejoOrganization); err != nil {
		panic(fmt.Errorf("UpdateOrganizationCols: %v %v", o.forgejoOrganization, err))
	}
}

func (o *organization) Put(ctx context.Context) f3_id.NodeID {
	node := o.GetNode()
	o.Trace("%s", node.GetID())

	doer, err := user_model.GetAdminUser(ctx)
	if err != nil {
		panic(fmt.Errorf("GetAdminUser %w", err))
	}
	err = org_model.CreateOrganization(ctx, o.forgejoOrganization, doer)
	if err != nil {
		panic(err)
	}

	return f3_id.NewNodeID(o.forgejoOrganization.ID)
}

func (o *organization) Delete(ctx context.Context) {
	node := o.GetNode()
	o.Trace("%s", node.GetID())

	if err := org_model.DeleteOrganization(ctx, o.forgejoOrganization); err != nil {
		panic(err)
	}
}

func newOrganization() generic.NodeDriverInterface {
	return &organization{}
}
