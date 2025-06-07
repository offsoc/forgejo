// Copyright 2024 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"net/http"

	"forgejo.org/models/unit"
	"forgejo.org/modules/optional"
	"forgejo.org/services/context"
	issue_service "forgejo.org/services/issue"
)

// IssueSuggestions returns a list of issue suggestions
func IssueSuggestions(ctx *context.Context) {
	canReadIssues := ctx.Repo.CanRead(unit.TypeIssues)
	canReadPulls := ctx.Repo.CanRead(unit.TypePullRequests)

	var isPull optional.Option[bool]
	if canReadPulls && !canReadIssues {
		isPull = optional.Some(true)
	} else if canReadIssues && !canReadPulls {
		isPull = optional.Some(false)
	}

	suggestions, err := issue_service.GetSuggestions(ctx, ctx.Repo.Repository, isPull)
	if err != nil {
		ctx.ServerError("GetSuggestions", err)
		return
	}

	ctx.JSON(http.StatusOK, suggestions)
}
