// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package org

import (
	"forgejo.org/routers/api/v1/shared"
	"forgejo.org/services/context"
)

// GetQuota returns the quota information for a given organization
func GetQuota(ctx *context.APIContext) {
	// swagger:operation GET /orgs/{org}/quota organization orgGetQuota
	// ---
	// summary: Get quota information for an organization
	// produces:
	// - application/json
	// parameters:
	// - name: org
	//   in: path
	//   description: name of the organization
	//   type: string
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/QuotaInfo"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"

	shared.GetQuota(ctx, ctx.Org.Organization.ID)
}

// CheckQuota returns whether the organization in context is over the subject quota
func CheckQuota(ctx *context.APIContext) {
	// swagger:operation GET /orgs/{org}/quota/check organization orgCheckQuota
	// ---
	// summary: Check if the organization is over quota for a given subject
	// produces:
	// - application/json
	// parameters:
	// - name: org
	//   in: path
	//   description: name of the organization
	//   type: string
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/boolean"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"
	//   "422":
	//     "$ref": "#/responses/validationError"

	shared.CheckQuota(ctx, ctx.Org.Organization.ID)
}

// ListQuotaAttachments lists attachments affecting the organization's quota
func ListQuotaAttachments(ctx *context.APIContext) {
	// swagger:operation GET /orgs/{org}/quota/attachments organization orgListQuotaAttachments
	// ---
	// summary: List the attachments affecting the organization's quota
	// produces:
	// - application/json
	// parameters:
	// - name: org
	//   in: path
	//   description: name of the organization
	//   type: string
	//   required: true
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
	//   "404":
	//     "$ref": "#/responses/notFound"

	shared.ListQuotaAttachments(ctx, ctx.Org.Organization.ID)
}

// ListQuotaPackages lists packages affecting the organization's quota
func ListQuotaPackages(ctx *context.APIContext) {
	// swagger:operation GET /orgs/{org}/quota/packages organization orgListQuotaPackages
	// ---
	// summary: List the packages affecting the organization's quota
	// produces:
	// - application/json
	// parameters:
	// - name: org
	//   in: path
	//   description: name of the organization
	//   type: string
	//   required: true
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
	//   "404":
	//     "$ref": "#/responses/notFound"

	shared.ListQuotaPackages(ctx, ctx.Org.Organization.ID)
}

// ListQuotaArtifacts lists artifacts affecting the organization's quota
func ListQuotaArtifacts(ctx *context.APIContext) {
	// swagger:operation GET /orgs/{org}/quota/artifacts organization orgListQuotaArtifacts
	// ---
	// summary: List the artifacts affecting the organization's quota
	// produces:
	// - application/json
	// parameters:
	// - name: org
	//   in: path
	//   description: name of the organization
	//   type: string
	//   required: true
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
	//   "404":
	//     "$ref": "#/responses/notFound"

	shared.ListQuotaArtifacts(ctx, ctx.Org.Organization.ID)
}
