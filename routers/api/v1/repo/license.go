// Copyright 2024 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"net/http"

	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/services/context"
	"code.gitea.io/gitea/services/convert"
)

// GetLicenses returns licenses
func GetLicenses(ctx *context.APIContext) {
	// swagger:operation GET /repos/{owner}/{repo}/licenses repository repoGetLicenses
	// ---
	// summary: Get repo licenses
	// produces:
	//   - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// responses:
	//   "404":
	//     "$ref": "#/responses/notFound"
	//   "200":
	//     "$ref": "#/responses/LicensesList"

	licenses, err := repo_model.GetRepoLicenses(ctx, ctx.Repo.Repository)
	if err != nil {
		log.Error("GetRepoLicenses failed: %v", err)
		ctx.InternalServerError(err)
		return
	}

	ctx.JSON(http.StatusOK, convert.ToRepoLicenseList(licenses))
}
