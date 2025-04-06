// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package admin

import (
	"forgejo.org/routers/api/v1/shared"
	"forgejo.org/services/context"
)

// https://docs.github.com/en/rest/actions/self-hosted-runners?apiVersion=2022-11-28#create-a-registration-token-for-an-organization

// GetRegistrationToken returns the token to register global runners
func GetRegistrationToken(ctx *context.APIContext) {
	// swagger:operation GET /admin/runners/registration-token admin adminGetRunnerRegistrationToken
	// ---
	// summary: Get an global actions runner registration token
	// produces:
	// - application/json
	// parameters:
	// responses:
	//   "200":
	//     "$ref": "#/responses/RegistrationToken"

	shared.GetRegistrationToken(ctx, 0, 0)
}

// SearchActionRunJobs return a list of actions jobs filtered by the provided parameters
func SearchActionRunJobs(ctx *context.APIContext) {
	// swagger:operation GET /admin/runners/jobs admin adminSearchRunJobs
	// ---
	// summary: Search action jobs according filter conditions
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
	//   "403":
	//     "$ref": "#/responses/forbidden"
	shared.GetActionRunJobs(ctx, 0, 0)
}
