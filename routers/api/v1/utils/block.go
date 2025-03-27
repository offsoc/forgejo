// Copyright 2023 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package utils

import (
	"net/http"

	user_model "forgejo.org/models/user"
	api "forgejo.org/modules/structs"
	"forgejo.org/services/context"
	user_service "forgejo.org/services/user"
)

// ListUserBlockedUsers lists the blocked users of the provided doer.
func ListUserBlockedUsers(ctx *context.APIContext, doer *user_model.User) {
	count, err := user_model.CountBlockedUsers(ctx, doer.ID)
	if err != nil {
		ctx.InternalServerError(err)
		return
	}

	blockedUsers, err := user_model.ListBlockedUsers(ctx, doer.ID, GetListOptions(ctx))
	if err != nil {
		ctx.InternalServerError(err)
		return
	}

	apiBlockedUsers := make([]*api.BlockedUser, len(blockedUsers))
	for i, blockedUser := range blockedUsers {
		apiBlockedUsers[i] = &api.BlockedUser{
			BlockID: blockedUser.ID,
			Created: blockedUser.CreatedUnix.AsTime(),
		}
		if err != nil {
			ctx.InternalServerError(err)
			return
		}
	}

	ctx.SetTotalCountHeader(count)
	ctx.JSON(http.StatusOK, apiBlockedUsers)
}

// BlockUser blocks the blockUser from the doer.
func BlockUser(ctx *context.APIContext, doer, blockUser *user_model.User) {
	err := user_service.BlockUser(ctx, doer.ID, blockUser.ID)
	if err != nil {
		ctx.InternalServerError(err)
		return
	}

	ctx.Status(http.StatusNoContent)
}

// UnblockUser unblocks the blockUser from the doer.
func UnblockUser(ctx *context.APIContext, doer, blockUser *user_model.User) {
	err := user_model.UnblockUser(ctx, doer.ID, blockUser.ID)
	if err != nil {
		ctx.InternalServerError(err)
		return
	}

	ctx.Status(http.StatusNoContent)
}
