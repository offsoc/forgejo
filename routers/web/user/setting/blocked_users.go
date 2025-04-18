// Copyright 2023 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package setting

import (
	"net/http"

	"forgejo.org/models/db"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/base"
	"forgejo.org/modules/setting"
	"forgejo.org/services/context"
)

const (
	tplSettingsBlockedUsers base.TplName = "user/settings/blocked_users"
)

// BlockedUsers render the blocked users list page.
func BlockedUsers(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("settings.blocked_users")
	ctx.Data["PageIsBlockedUsers"] = true
	ctx.Data["BaseLink"] = setting.AppSubURL + "/user/settings/blocked_users"
	ctx.Data["BaseLinkNew"] = setting.AppSubURL + "/user/settings/blocked_users"

	blockedUsers, err := user_model.ListBlockedUsers(ctx, ctx.Doer.ID, db.ListOptions{})
	if err != nil {
		ctx.ServerError("ListBlockedUsers", err)
		return
	}

	ctx.Data["BlockedUsers"] = blockedUsers
	ctx.HTML(http.StatusOK, tplSettingsBlockedUsers)
}

// UnblockUser unblocks a particular user for the doer.
func UnblockUser(ctx *context.Context) {
	if err := user_model.UnblockUser(ctx, ctx.Doer.ID, ctx.FormInt64("user_id")); err != nil {
		ctx.ServerError("UnblockUser", err)
		return
	}

	ctx.Flash.Success(ctx.Tr("settings.user_unblock_success"))
	ctx.Redirect(setting.AppSubURL + "/user/settings/blocked_users")
}
