// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package federation

import (
	"fmt"
	"net/http"

	"code.gitea.io/gitea/models/forgefed"
	"code.gitea.io/gitea/modules/log"
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
		following, err := forgefed.IsFollowing(ctx, ctx.ContextUser.ID, federatedUser.ID)
		if err != nil {
			log.Error("forgefed.IsFollowing: %v", err)
			ctx.Error(http.StatusInternalServerError, "forgefed.IsFollowing", err)
			return
		}
		if !following {
			// The local user is not following the federated one, nothing to do.
			log.Trace("Local user[%d] is not following federated user[%d]", ctx.ContextUser.ID, federatedUser.ID)
			return
		}
		if err := forgefed.RemoveFollower(ctx, ctx.ContextUser.ID, federatedUser.ID); err != nil {
			ctx.Error(http.StatusInternalServerError, "Unable to remove follower", err)
			return
		}
	}

	ctx.Status(http.StatusNoContent)
}
