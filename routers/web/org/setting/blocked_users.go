// Copyright 2023 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package setting

import (
	"fmt"
	"net/http"
	"strings"

	"forgejo.org/models/db"
	user_model "forgejo.org/models/user"
	shared_user "forgejo.org/routers/web/shared/user"
	"forgejo.org/services/context"
	user_service "forgejo.org/services/user"
)

const tplBlockedUsers = "org/settings/blocked_users"

// BlockedUsers renders the blocked users page.
func BlockedUsers(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("settings.blocked_users")
	ctx.Data["PageIsSettingsBlockedUsers"] = true

	blockedUsers, err := user_model.ListBlockedUsers(ctx, ctx.Org.Organization.ID, db.ListOptions{})
	if err != nil {
		ctx.ServerError("ListBlockedUsers", err)
		return
	}

	err = shared_user.LoadHeaderCount(ctx)
	if err != nil {
		ctx.ServerError("LoadHeaderCount", err)
		return
	}

	ctx.Data["BlockedUsers"] = blockedUsers

	ctx.HTML(http.StatusOK, tplBlockedUsers)
}

// BlockedUsersBlock blocks a particular user from the organization.
func BlockedUsersBlock(ctx *context.Context) {
	uname := strings.ToLower(ctx.FormString("uname"))
	u, err := user_model.GetUserByName(ctx, uname)
	if err != nil {
		ctx.ServerError("GetUserByName", err)
		return
	}

	if u.IsOrganization() {
		ctx.ServerError("IsOrganization", fmt.Errorf("%s is an organization not a user", u.Name))
		return
	}

	if u.ID == ctx.Doer.ID {
		ctx.Flash.Error(ctx.Tr("settings.user_block_yourself"))
		ctx.Redirect(ctx.Org.OrgLink + "/settings/blocked_users")
		return
	}

	if err := user_service.BlockUser(ctx, ctx.Org.Organization.ID, u.ID); err != nil {
		ctx.ServerError("BlockUser", err)
		return
	}

	ctx.Flash.Success(ctx.Tr("settings.user_block_success"))
	ctx.Redirect(ctx.Org.OrgLink + "/settings/blocked_users")
}

// BlockedUsersUnblock unblocks a particular user from the organization.
func BlockedUsersUnblock(ctx *context.Context) {
	u, err := user_model.GetUserByID(ctx, ctx.FormInt64("user_id"))
	if err != nil {
		ctx.ServerError("GetUserByID", err)
		return
	}

	if u.IsOrganization() {
		ctx.ServerError("IsOrganization", fmt.Errorf("%s is an organization not a user", u.Name))
		return
	}

	if err := user_model.UnblockUser(ctx, ctx.Org.Organization.ID, u.ID); err != nil {
		ctx.ServerError("UnblockUser", err)
		return
	}

	ctx.Flash.Success(ctx.Tr("settings.user_unblock_success"))
	ctx.Redirect(ctx.Org.OrgLink + "/settings/blocked_users")
}
