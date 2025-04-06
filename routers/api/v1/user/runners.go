// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user

import (
	"forgejo.org/routers/api/v1/shared"
	"forgejo.org/services/context"
)

// https://docs.github.com/en/rest/actions/self-hosted-runners?apiVersion=2022-11-28#create-a-registration-token-for-an-organization

// GetRegistrationToken returns the token to register user runners
func GetRegistrationToken(ctx *context.APIContext) {
	// swagger:operation GET /user/actions/runners/registration-token user userGetRunnerRegistrationToken
	// ---
	// summary: Get an user's actions runner registration token
	// produces:
	// - application/json
	// parameters:
	// responses:
	//   "200":
	//     "$ref": "#/responses/RegistrationToken"
	//   "401":
	//     "$ref": "#/responses/unauthorized"
	//   "403":
	//     "$ref": "#/responses/forbidden"

	shared.GetRegistrationToken(ctx, ctx.Doer.ID, 0)
}

// SearchActionRunJobs return a list of actions jobs filtered by the provided parameters
func SearchActionRunJobs(ctx *context.APIContext) {
	// swagger:operation GET /user/actions/runners/jobs user userSearchRunJobs
	// ---
	// summary: Search for user's action jobs according filter conditions
	// produces:
	// - application/json
	// parameters:
	// - name: labels
	//   in: query
	//   description: a comma separated list of run job labels to search for
	//   type: string
	// responses:
	//   "200":
	//     "$ref": "#/responses/RunJobList"
	//   "401":
	//     "$ref": "#/responses/unauthorized"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	shared.GetActionRunJobs(ctx, ctx.Doer.ID, 0)
}
