// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user

import (
	"code.gitea.io/gitea/routers/api/v1/shared"
	"code.gitea.io/gitea/services/context"
)

// GetQuota returns the quota information for the authenticated user
func GetQuota(ctx *context.APIContext) {
	// swagger:operation GET /user/quota user userGetQuota
	// ---
	// summary: Get quota information for the authenticated user
	// produces:
	// - application/json
	// responses:
	//   "200":
	//     "$ref": "#/responses/QuotaInfo"
	//   "403":
	//     "$ref": "#/responses/forbidden"

	shared.GetQuota(ctx, ctx.Doer.ID)
}

// CheckQuota returns whether the authenticated user is over the subject quota
func CheckQuota(ctx *context.APIContext) {
	// swagger:operation GET /user/quota/check user userCheckQuota
	// ---
	// summary: Check if the authenticated user is over quota for a given subject
	// produces:
	// - application/json
	// responses:
	//   "200":
	//     "$ref": "#/responses/boolean"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "422":
	//     "$ref": "#/responses/validationError"

	shared.CheckQuota(ctx, ctx.Doer.ID)
}

// ListQuotaAttachments lists attachments affecting the authenticated user's quota
func ListQuotaAttachments(ctx *context.APIContext) {
	// swagger:operation GET /user/quota/attachments user userListQuotaAttachments
	// ---
	// summary: List the attachments affecting the authenticated user's quota
	// produces:
	// - application/json
	// parameters:
	// - name: page
	//   in: query
	//   description: page number of results to return (1-based)
	//   type: integer
	// - name: limit
	//   in: query
	//   description: page size of results
	//   type: integer
	// responses:
	//   "200":
	//     "$ref": "#/responses/QuotaUsedAttachmentList"
	//   "403":
	//     "$ref": "#/responses/forbidden"

	shared.ListQuotaAttachments(ctx, ctx.Doer.ID)
}

// ListQuotaPackages lists packages affecting the authenticated user's quota
func ListQuotaPackages(ctx *context.APIContext) {
	// swagger:operation GET /user/quota/packages user userListQuotaPackages
	// ---
	// summary: List the packages affecting the authenticated user's quota
	// produces:
	// - application/json
	// parameters:
	// - name: page
	//   in: query
	//   description: page number of results to return (1-based)
	//   type: integer
	// - name: limit
	//   in: query
	//   description: page size of results
	//   type: integer
	// responses:
	//   "200":
	//     "$ref": "#/responses/QuotaUsedPackageList"
	//   "403":
	//     "$ref": "#/responses/forbidden"

	shared.ListQuotaPackages(ctx, ctx.Doer.ID)
}

// ListQuotaArtifacts lists artifacts affecting the authenticated user's quota
func ListQuotaArtifacts(ctx *context.APIContext) {
	// swagger:operation GET /user/quota/artifacts user userListQuotaArtifacts
	// ---
	// summary: List the artifacts affecting the authenticated user's quota
	// produces:
	// - application/json
	// parameters:
	// - name: page
	//   in: query
	//   description: page number of results to return (1-based)
	//   type: integer
	// - name: limit
	//   in: query
	//   description: page size of results
	//   type: integer
	// responses:
	//   "200":
	//     "$ref": "#/responses/QuotaUsedArtifactList"
	//   "403":
	//     "$ref": "#/responses/forbidden"

	shared.ListQuotaArtifacts(ctx, ctx.Doer.ID)
}
