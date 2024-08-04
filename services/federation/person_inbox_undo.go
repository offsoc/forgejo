// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package federation

import (
	"fmt"
	"net/http"

	"code.gitea.io/gitea/models/forgefed"
	context_service "code.gitea.io/gitea/services/context"

	ap "github.com/go-ap/activitypub"
)

func processPersonInboxUndo(ctx *context_service.APIContext, activity *ap.Activity) {
	if activity.Object.GetType() != ap.FollowType {
		ctx.Error(http.StatusNotAcceptable, "Invalid object type for Undo activity", fmt.Errorf("Invalid object type for Undo activity: %v", activity.Object.GetType()))
		return
	}

	actorURI := activity.Actor.GetLink().String()
	_, federatedUser, _, _, err := findFederatedUser(ctx, actorURI)
	if err != nil {
		return
	}

	if federatedUser != nil {
		if err := forgefed.RemoveFollower(ctx, ctx.ContextUser.ID, federatedUser.ID); err != nil {
			ctx.Error(http.StatusInternalServerError, "Unable to remove follower", err)
			return
		}
	}

	ctx.Status(http.StatusNoContent)
}
