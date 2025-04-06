// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user

import (
	"net/http"

	"forgejo.org/modules/optional"
	api "forgejo.org/modules/structs"
	"forgejo.org/modules/web"
	"forgejo.org/services/context"
	"forgejo.org/services/convert"
	user_service "forgejo.org/services/user"
)

// GetUserSettings returns user settings
func GetUserSettings(ctx *context.APIContext) {
	// swagger:operation GET /user/settings user getUserSettings
	// ---
	// summary: Get user settings
	// produces:
	// - application/json
	// responses:
	//   "200":
	//     "$ref": "#/responses/UserSettings"
	//   "401":
	//     "$ref": "#/responses/unauthorized"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	ctx.JSON(http.StatusOK, convert.User2UserSettings(ctx.Doer))
}

// UpdateUserSettings returns user settings
func UpdateUserSettings(ctx *context.APIContext) {
	// swagger:operation PATCH /user/settings user updateUserSettings
	// ---
	// summary: Update user settings
	// parameters:
	// - name: body
	//   in: body
	//   schema:
	//     "$ref": "#/definitions/UserSettingsOptions"
	// produces:
	// - application/json
	// responses:
	//   "200":
	//     "$ref": "#/responses/UserSettings"
	//   "401":
	//     "$ref": "#/responses/unauthorized"
	//   "403":
	//     "$ref": "#/responses/forbidden"

	form := web.GetForm(ctx).(*api.UserSettingsOptions)

	opts := &user_service.UpdateOptions{
		FullName:            optional.FromPtr(form.FullName),
		Description:         optional.FromPtr(form.Description),
		Pronouns:            optional.FromPtr(form.Pronouns),
		Website:             optional.FromPtr(form.Website),
		Location:            optional.FromPtr(form.Location),
		Language:            optional.FromPtr(form.Language),
		Theme:               optional.FromPtr(form.Theme),
		DiffViewStyle:       optional.FromPtr(form.DiffViewStyle),
		KeepEmailPrivate:    optional.FromPtr(form.HideEmail),
		KeepPronounsPrivate: optional.FromPtr(form.HidePronouns),
		KeepActivityPrivate: optional.FromPtr(form.HideActivity),
		EnableRepoUnitHints: optional.FromPtr(form.EnableRepoUnitHints),
	}
	if err := user_service.UpdateUser(ctx, ctx.Doer, opts); err != nil {
		ctx.InternalServerError(err)
		return
	}

	ctx.JSON(http.StatusOK, convert.User2UserSettings(ctx.Doer))
}
