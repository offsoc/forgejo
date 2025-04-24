// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package federation

import (
	"net/http"

	"forgejo.org/models/forgefed"
	fm "forgejo.org/modules/forgefed"
	"forgejo.org/modules/log"
	context_service "forgejo.org/services/context"

	ap "github.com/go-ap/activitypub"
)

func processPersonInboxCreate(ctx *context_service.APIContext, activity *ap.Activity) {
	createAct, err := fm.NewForgeUserActivityFromAp(*activity)
	if err != nil {
		log.Error("Invalid user activity: %v, %v", activity, err)
		ctx.Error(http.StatusNotAcceptable, "Invalid user activity", err)
		return
	}

	actorURI := createAct.Actor.GetLink().String()
	if _, _, _, err := findFederatedUser(ctx, actorURI); err != nil {
		log.Error("Error finding federated user (%s): %v", actorURI, err)
		ctx.Error(http.StatusNotAcceptable, "Federated user not found", err)
		return
	}

	if err := forgefed.AddUserActivity(ctx, ctx.ContextUser.ID, actorURI, createAct.Note); err != nil {
		log.Error("Unable to record activity: %v", err)
		ctx.Error(http.StatusInternalServerError, "Unable to record activity", err)
		return
	}

	ctx.Status(http.StatusNoContent)
}
