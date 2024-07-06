// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package context

import (
	"net/http"

	quota_model "code.gitea.io/gitea/models/quota"
)

// QuotaGroupAssignmentAPI returns a middleware to handle context-quota-group assignment for api routes
func QuotaGroupAssignmentAPI() func(ctx *APIContext) {
	return func(ctx *APIContext) {
		groupName := ctx.Params("quotagroup")
		group, err := quota_model.GetGroupByName(ctx, groupName)
		if err != nil {
			ctx.Error(http.StatusInternalServerError, "quota_model.GetGroupByName", err)
			return
		}
		if group == nil {
			ctx.NotFound()
			return
		}
		ctx.QuotaGroup = group
	}
}

// QuotaRuleAssignmentAPI returns a middleware to handle context-quota-rule assignment for api routes
func QuotaRuleAssignmentAPI() func(ctx *APIContext) {
	return func(ctx *APIContext) {
		ruleName := ctx.Params("quotarule")
		rule, err := quota_model.GetRuleByName(ctx, ruleName)
		if err != nil {
			ctx.Error(http.StatusInternalServerError, "quota_model.GetRuleByName", err)
			return
		}
		if rule == nil {
			ctx.NotFound()
			return
		}
		ctx.QuotaRule = rule
	}
}
