// Copyright 2023, 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package context

import (
	"net/http"

	repo_model "forgejo.org/models/repo"
)

// RepositoryIDAssignmentAPI returns a middleware to handle context-repo assignment for api routes
func RepositoryIDAssignmentAPI() func(ctx *APIContext) {
	return func(ctx *APIContext) {
		repositoryID := ctx.ParamsInt64(":repository-id")

		var err error
		repository := new(Repository)
		repository.Repository, err = repo_model.GetRepositoryByID(ctx, repositoryID)
		if err != nil {
			ctx.Error(http.StatusNotFound, "GetRepositoryByID", err)
		}
		ctx.Repo = repository
	}
}
