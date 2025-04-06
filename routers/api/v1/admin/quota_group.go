// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package admin

import (
	go_context "context"
	"net/http"

	"forgejo.org/models/db"
	quota_model "forgejo.org/models/quota"
	api "forgejo.org/modules/structs"
	"forgejo.org/modules/web"
	"forgejo.org/services/context"
	"forgejo.org/services/convert"
)

// ListQuotaGroups returns all the quota groups
func ListQuotaGroups(ctx *context.APIContext) {
	// swagger:operation GET /admin/quota/groups admin adminListQuotaGroups
	// ---
	// summary: List the available quota groups
	// produces:
	// - application/json
	// responses:
	//   "200":
	//     "$ref": "#/responses/QuotaGroupList"
	//   "403":
	//     "$ref": "#/responses/forbidden"

	groups, err := quota_model.ListGroups(ctx)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "quota_model.ListGroups", err)
		return
	}
	for _, group := range groups {
		if err = group.LoadRules(ctx); err != nil {
			ctx.Error(http.StatusInternalServerError, "quota_model.group.LoadRules", err)
			return
		}
	}

	ctx.JSON(http.StatusOK, convert.ToQuotaGroupList(groups, true))
}

func createQuotaGroupWithRules(ctx go_context.Context, opts *api.CreateQuotaGroupOptions) (*quota_model.Group, error) {
	ctx, committer, err := db.TxContext(ctx)
	if err != nil {
		return nil, err
	}
	defer committer.Close()

	group, err := quota_model.CreateGroup(ctx, opts.Name)
	if err != nil {
		return nil, err
	}

	for _, rule := range opts.Rules {
		exists, err := quota_model.DoesRuleExist(ctx, rule.Name)
		if err != nil {
			return nil, err
		}
		if !exists {
			var limit int64
			if rule.Limit != nil {
				limit = *rule.Limit
			}

			subjects, err := toLimitSubjects(rule.Subjects)
			if err != nil {
				return nil, err
			}

			_, err = quota_model.CreateRule(ctx, rule.Name, limit, *subjects)
			if err != nil {
				return nil, err
			}
		}
		if err = group.AddRuleByName(ctx, rule.Name); err != nil {
			return nil, err
		}
	}

	if err = group.LoadRules(ctx); err != nil {
		return nil, err
	}

	return group, committer.Commit()
}

// CreateQuotaGroup creates a new quota group
func CreateQuotaGroup(ctx *context.APIContext) {
	// swagger:operation POST /admin/quota/groups admin adminCreateQuotaGroup
	// ---
	// summary: Create a new quota group
	// produces:
	// - application/json
	// parameters:
	// - name: group
	//   in: body
	//   description: Definition of the quota group
	//   schema:
	//     "$ref": "#/definitions/CreateQuotaGroupOptions"
	//   required: true
	// responses:
	//   "201":
	//     "$ref": "#/responses/QuotaGroup"
	//   "400":
	//     "$ref": "#/responses/error"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "409":
	//     "$ref": "#/responses/error"
	//   "422":
	//     "$ref": "#/responses/validationError"

	form := web.GetForm(ctx).(*api.CreateQuotaGroupOptions)

	group, err := createQuotaGroupWithRules(ctx, form)
	if err != nil {
		if quota_model.IsErrGroupAlreadyExists(err) {
			ctx.Error(http.StatusConflict, "", err)
		} else if quota_model.IsErrParseLimitSubjectUnrecognized(err) {
			ctx.Error(http.StatusUnprocessableEntity, "", err)
		} else {
			ctx.Error(http.StatusInternalServerError, "quota_model.CreateGroup", err)
		}
		return
	}
	ctx.JSON(http.StatusCreated, convert.ToQuotaGroup(*group, true))
}

// ListUsersInQuotaGroup lists all the users in a quota group
func ListUsersInQuotaGroup(ctx *context.APIContext) {
	// swagger:operation GET /admin/quota/groups/{quotagroup}/users admin adminListUsersInQuotaGroup
	// ---
	// summary: List users in a quota group
	// produces:
	// - application/json
	// parameters:
	// - name: quotagroup
	//   in: path
	//   description: quota group to list members of
	//   type: string
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/UserList"
	//   "400":
	//     "$ref": "#/responses/error"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"

	users, err := quota_model.ListUsersInGroup(ctx, ctx.QuotaGroup.Name)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "quota_model.ListUsersInGroup", err)
		return
	}
	ctx.JSON(http.StatusOK, convert.ToUsers(ctx, ctx.Doer, users))
}

// AddUserToQuotaGroup adds a user to a quota group
func AddUserToQuotaGroup(ctx *context.APIContext) {
	// swagger:operation PUT /admin/quota/groups/{quotagroup}/users/{username} admin adminAddUserToQuotaGroup
	// ---
	// summary: Add a user to a quota group
	// produces:
	// - application/json
	// parameters:
	// - name: quotagroup
	//   in: path
	//   description: quota group to add the user to
	//   type: string
	//   required: true
	// - name: username
	//   in: path
	//   description: username of the user to add to the quota group
	//   type: string
	//   required: true
	// responses:
	//   "204":
	//     "$ref": "#/responses/empty"
	//   "400":
	//     "$ref": "#/responses/error"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"
	//   "409":
	//     "$ref": "#/responses/error"
	//   "422":
	//     "$ref": "#/responses/validationError"

	err := ctx.QuotaGroup.AddUserByID(ctx, ctx.ContextUser.ID)
	if err != nil {
		if quota_model.IsErrUserAlreadyInGroup(err) {
			ctx.Error(http.StatusConflict, "", err)
		} else {
			ctx.Error(http.StatusInternalServerError, "quota_group.group.AddUserByID", err)
		}
		return
	}
	ctx.Status(http.StatusNoContent)
}

// RemoveUserFromQuotaGroup removes a user from a quota group
func RemoveUserFromQuotaGroup(ctx *context.APIContext) {
	// swagger:operation DELETE /admin/quota/groups/{quotagroup}/users/{username} admin adminRemoveUserFromQuotaGroup
	// ---
	// summary: Remove a user from a quota group
	// produces:
	// - application/json
	// parameters:
	// - name: quotagroup
	//   in: path
	//   description: quota group to remove a user from
	//   type: string
	//   required: true
	// - name: username
	//   in: path
	//   description: username of the user to remove from the quota group
	//   type: string
	//   required: true
	// responses:
	//   "204":
	//     "$ref": "#/responses/empty"
	//   "400":
	//     "$ref": "#/responses/error"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"

	err := ctx.QuotaGroup.RemoveUserByID(ctx, ctx.ContextUser.ID)
	if err != nil {
		if quota_model.IsErrUserNotInGroup(err) {
			ctx.NotFound()
		} else {
			ctx.Error(http.StatusInternalServerError, "quota_model.group.RemoveUserByID", err)
		}
		return
	}
	ctx.Status(http.StatusNoContent)
}

// SetUserQuotaGroups moves the user to specific quota groups
func SetUserQuotaGroups(ctx *context.APIContext) {
	// swagger:operation POST /admin/users/{username}/quota/groups admin adminSetUserQuotaGroups
	// ---
	// summary: Set the user's quota groups to a given list.
	// produces:
	// - application/json
	// parameters:
	// - name: username
	//   in: path
	//   description: username of the user to modify the quota groups from
	//   type: string
	//   required: true
	// - name: groups
	//   in: body
	//   description: list of groups that the user should be a member of
	//   schema:
	//     "$ref": "#/definitions/SetUserQuotaGroupsOptions"
	//   required: true
	// responses:
	//   "204":
	//     "$ref": "#/responses/empty"
	//   "400":
	//     "$ref": "#/responses/error"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"
	//   "422":
	//     "$ref": "#/responses/validationError"

	form := web.GetForm(ctx).(*api.SetUserQuotaGroupsOptions)

	err := quota_model.SetUserGroups(ctx, ctx.ContextUser.ID, form.Groups)
	if err != nil {
		if quota_model.IsErrGroupNotFound(err) {
			ctx.Error(http.StatusUnprocessableEntity, "", err)
		} else {
			ctx.Error(http.StatusInternalServerError, "quota_model.SetUserGroups", err)
		}
		return
	}

	ctx.Status(http.StatusNoContent)
}

// DeleteQuotaGroup deletes a quota group
func DeleteQuotaGroup(ctx *context.APIContext) {
	// swagger:operation DELETE /admin/quota/groups/{quotagroup} admin adminDeleteQuotaGroup
	// ---
	// summary: Delete a quota group
	// produces:
	// - application/json
	// parameters:
	// - name: quotagroup
	//   in: path
	//   description: quota group to delete
	//   type: string
	//   required: true
	// responses:
	//   "204":
	//     "$ref": "#/responses/empty"
	//   "400":
	//     "$ref": "#/responses/error"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"

	err := quota_model.DeleteGroupByName(ctx, ctx.QuotaGroup.Name)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "quota_model.DeleteGroupByName", err)
		return
	}

	ctx.Status(http.StatusNoContent)
}

// GetQuotaGroup returns information about a quota group
func GetQuotaGroup(ctx *context.APIContext) {
	// swagger:operation GET /admin/quota/groups/{quotagroup} admin adminGetQuotaGroup
	// ---
	// summary: Get information about the quota group
	// produces:
	// - application/json
	// parameters:
	// - name: quotagroup
	//   in: path
	//   description: quota group to query
	//   type: string
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/QuotaGroup"
	//   "400":
	//     "$ref": "#/responses/error"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"

	ctx.JSON(http.StatusOK, convert.ToQuotaGroup(*ctx.QuotaGroup, true))
}

// AddRuleToQuotaGroup adds a rule to a quota group
func AddRuleToQuotaGroup(ctx *context.APIContext) {
	// swagger:operation PUT /admin/quota/groups/{quotagroup}/rules/{quotarule} admin adminAddRuleToQuotaGroup
	// ---
	// summary: Adds a rule to a quota group
	// produces:
	// - application/json
	// parameters:
	// - name: quotagroup
	//   in: path
	//   description: quota group to add a rule to
	//   type: string
	//   required: true
	// - name: quotarule
	//   in: path
	//   description: the name of the quota rule to add to the group
	//   type: string
	//   required: true
	// responses:
	//   "204":
	//     "$ref": "#/responses/empty"
	//   "400":
	//     "$ref": "#/responses/error"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"
	//   "409":
	//     "$ref": "#/responses/error"
	//   "422":
	//     "$ref": "#/responses/validationError"

	err := ctx.QuotaGroup.AddRuleByName(ctx, ctx.QuotaRule.Name)
	if err != nil {
		if quota_model.IsErrRuleAlreadyInGroup(err) {
			ctx.Error(http.StatusConflict, "", err)
		} else if quota_model.IsErrRuleNotFound(err) {
			ctx.Error(http.StatusUnprocessableEntity, "", err)
		} else {
			ctx.Error(http.StatusInternalServerError, "quota_model.group.AddRuleByName", err)
		}
		return
	}
	ctx.Status(http.StatusNoContent)
}

// RemoveRuleFromQuotaGroup removes a rule from a quota group
func RemoveRuleFromQuotaGroup(ctx *context.APIContext) {
	// swagger:operation DELETE /admin/quota/groups/{quotagroup}/rules/{quotarule} admin adminRemoveRuleFromQuotaGroup
	// ---
	// summary: Removes a rule from a quota group
	// produces:
	// - application/json
	// parameters:
	// - name: quotagroup
	//   in: path
	//   description: quota group to remove a rule from
	//   type: string
	//   required: true
	// - name: quotarule
	//   in: path
	//   description: the name of the quota rule to remove from the group
	//   type: string
	//   required: true
	// responses:
	//   "201":
	//     "$ref": "#/responses/empty"
	//   "400":
	//     "$ref": "#/responses/error"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"

	err := ctx.QuotaGroup.RemoveRuleByName(ctx, ctx.QuotaRule.Name)
	if err != nil {
		if quota_model.IsErrRuleNotInGroup(err) {
			ctx.NotFound()
		} else {
			ctx.Error(http.StatusInternalServerError, "quota_model.group.RemoveRuleByName", err)
		}
		return
	}
	ctx.Status(http.StatusNoContent)
}
