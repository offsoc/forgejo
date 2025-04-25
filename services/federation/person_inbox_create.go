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
	user, _, _, err := findFederatedUser(ctx, actorURI)
	if err != nil {
		log.Error("Error finding federated user (%s): %v", actorURI, err)
		ctx.Error(http.StatusNotAcceptable, "Error finding federated user", err)
		return
	}

	federatedUserActivity, err := forgefed.NewFederatedUserActivity(
		user.ID, actorURI,
		createAct.Note.Content.String(),
		createAct.Note.URL.GetID().String(),
		*activity,
	)
	if err != nil {
		log.Error("Error creating federatedUserActivity (%s): %v", actorURI, err)
		ctx.Error(http.StatusNotAcceptable, "Error creating federatedUserActivity", err)
		return
	}

	if err := forgefed.CreateUserActivity(ctx, &federatedUserActivity); err != nil {
		log.Error("Unable to record activity: %v", err)
		ctx.Error(http.StatusInternalServerError, "Unable to record activity", err)
		return
	}

	ctx.Status(http.StatusNoContent)
}
