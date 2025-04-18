// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user

import (
	"net/http"

	access_model "forgejo.org/models/perm/access"
	repo_model "forgejo.org/models/repo"
	user_model "forgejo.org/models/user"
	api "forgejo.org/modules/structs"
	"forgejo.org/routers/api/v1/utils"
	"forgejo.org/services/context"
	"forgejo.org/services/convert"
)

// listUserRepos - List the repositories owned by the given user.
func listUserRepos(ctx *context.APIContext, u *user_model.User, private bool) {
	opts := utils.GetListOptions(ctx)

	repos, count, err := repo_model.GetUserRepositories(ctx, &repo_model.SearchRepoOptions{
		Actor:       u,
		Private:     private,
		ListOptions: opts,
		OrderBy:     "id ASC",
	})
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "GetUserRepositories", err)
		return
	}

	if err := repos.LoadAttributes(ctx); err != nil {
		ctx.Error(http.StatusInternalServerError, "RepositoryList.LoadAttributes", err)
		return
	}

	apiRepos := make([]*api.Repository, 0, len(repos))
	for i := range repos {
		permission, err := access_model.GetUserRepoPermission(ctx, repos[i], ctx.Doer)
		if err != nil {
			ctx.Error(http.StatusInternalServerError, "GetUserRepoPermission", err)
			return
		}
		if ctx.IsSigned && ctx.Doer.IsAdmin || permission.HasAccess() {
			apiRepos = append(apiRepos, convert.ToRepo(ctx, repos[i], permission))
		}
	}

	ctx.SetLinkHeader(int(count), opts.PageSize)
	ctx.SetTotalCountHeader(count)
	ctx.JSON(http.StatusOK, &apiRepos)
}

// ListUserRepos - list the repos owned by the given user.
func ListUserRepos(ctx *context.APIContext) {
	// swagger:operation GET /users/{username}/repos user userListRepos
	// ---
	// summary: List the repos owned by the given user
	// produces:
	// - application/json
	// parameters:
	// - name: username
	//   in: path
	//   description: username of user
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
	//     "$ref": "#/responses/RepositoryList"
	//   "404":
	//     "$ref": "#/responses/notFound"

	private := ctx.IsSigned
	listUserRepos(ctx, ctx.ContextUser, private)
}

// ListMyRepos - list the repositories you own or have access to.
func ListMyRepos(ctx *context.APIContext) {
	// swagger:operation GET /user/repos user userCurrentListRepos
	// ---
	// summary: List the repos that the authenticated user owns
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
	// - name: order_by
	//   in: query
	//   description: order the repositories
	//   type: string
	//   enum: [name, id, newest, oldest, recentupdate, leastupdate, reversealphabetically, alphabetically, reversesize, size, reversegitsize, gitsize, reverselfssize, lfssize, moststars, feweststars, mostforks, fewestforks]
	// responses:
	//   "200":
	//     "$ref": "#/responses/RepositoryList"
	//   "401":
	//     "$ref": "#/responses/unauthorized"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "422":
	//     "$ref": "#/responses/validationError"

	opts := &repo_model.SearchRepoOptions{
		ListOptions:        utils.GetListOptions(ctx),
		Actor:              ctx.Doer,
		OwnerID:            ctx.Doer.ID,
		Private:            ctx.IsSigned,
		IncludeDescription: true,
	}
	orderBy := ctx.FormTrim("order_by")
	switch orderBy {
	case "name":
		opts.OrderBy = "name ASC"
	case "id":
		opts.OrderBy = "id ASC"
	default:
		if orderBy, ok := repo_model.OrderByFlatMap[orderBy]; ok {
			opts.OrderBy = orderBy
		} else if orderBy != "" {
			ctx.Error(http.StatusUnprocessableEntity, "", "invalid order_by")
			return
		}
	}

	repos, count, err := repo_model.SearchRepository(ctx, opts)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "SearchRepository", err)
		return
	}

	results := make([]*api.Repository, len(repos))
	for i, repo := range repos {
		if err = repo.LoadOwner(ctx); err != nil {
			ctx.Error(http.StatusInternalServerError, "LoadOwner", err)
			return
		}
		permission, err := access_model.GetUserRepoPermission(ctx, repo, ctx.Doer)
		if err != nil {
			ctx.Error(http.StatusInternalServerError, "GetUserRepoPermission", err)
		}
		results[i] = convert.ToRepo(ctx, repo, permission)
	}

	ctx.SetLinkHeader(int(count), opts.PageSize)
	ctx.SetTotalCountHeader(count)
	ctx.JSON(http.StatusOK, &results)
}

// ListOrgRepos - list the repositories of an organization.
func ListOrgRepos(ctx *context.APIContext) {
	// swagger:operation GET /orgs/{org}/repos organization orgListRepos
	// ---
	// summary: List an organization's repos
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
	//     "$ref": "#/responses/RepositoryList"
	//   "404":
	//     "$ref": "#/responses/notFound"

	listUserRepos(ctx, ctx.Org.Organization.AsUser(), ctx.IsSigned)
}
