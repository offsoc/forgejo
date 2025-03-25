// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package admin

import (
	"net/http"

	quota_model "forgejo.org/models/quota"
	"forgejo.org/services/context"
	"forgejo.org/services/convert"
)

// GetUserQuota return information about a user's quota
func GetUserQuota(ctx *context.APIContext) {
	// swagger:operation GET /admin/users/{username}/quota admin adminGetUserQuota
	// ---
	// summary: Get the user's quota info
	// produces:
	// - application/json
	// parameters:
	// - name: username
	//   in: path
	//   description: username of user to query
	//   type: string
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/QuotaInfo"
	//   "400":
	//     "$ref": "#/responses/error"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"
	//   "422":
	//     "$ref": "#/responses/validationError"

	used, err := quota_model.GetUsedForUser(ctx, ctx.ContextUser.ID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "quota_model.GetUsedForUser", err)
		return
	}

	groups, err := quota_model.GetGroupsForUser(ctx, ctx.ContextUser.ID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "quota_model.GetGroupsForUser", err)
		return
	}

	result := convert.ToQuotaInfo(used, groups, true)
	ctx.JSON(http.StatusOK, &result)
}
