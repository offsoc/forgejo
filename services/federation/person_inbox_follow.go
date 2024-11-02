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
	"github.com/go-ap/jsonld"
)

func processPersonFollow(ctx *context_service.APIContext, activity *ap.Activity) {
	if activity.Object.GetLink().String() != ctx.ContextUser.APActorID() {
		log.Error("User to follow does not match the inbox owner: %s != %s", activity.Object.GetLink().String(), ctx.ContextUser.APActorID())
		ctx.Error(http.StatusNotAcceptable, "Wrong user to follow", fmt.Errorf("User to follow does not match the inbox owner"))
		return
	}

	if activity.Actor.GetLink().String() == "" {
		log.Error("Activity is missing an actor: %#v", activity)
		ctx.Error(http.StatusNotAcceptable, "Missing actor", fmt.Errorf("Missing Actor"))
		return
	}

	actorURI := activity.Actor.GetLink().String()
	_, federatedUser, _, err := findOrCreateFederatedUser(ctx, actorURI)
	if err != nil {
		log.Error("Error finding or creating federated user (%s): %v", actorURI, err)
		ctx.Error(http.StatusNotAcceptable, "Federated user not found", err)
		return
	}

	following, err := forgefed.IsFollowing(ctx, ctx.ContextUser.ID, federatedUser.ID)
	if err != nil {
		log.Error("forgefed.IsFollowing: %v", err)
		ctx.Error(http.StatusInternalServerError, "forgefed.IsFollowing", err)
		return
	}
	if following {
		// If the user is already following, we're good, nothing to do.
		log.Trace("Local user[%d] is already following federated user[%d]", ctx.ContextUser.ID, federatedUser.ID)
		return
	}

	followingID, err := forgefed.AddFollower(ctx, ctx.ContextUser.ID, federatedUser.ID)
	if err != nil {
		log.Error("Unable to add follower: %v", err)
		ctx.Error(http.StatusInternalServerError, "Unable to add follower", err)
		return
	}

	// Respond back with an accept
	binary := []byte(`{"status":"Accepted"}`)
	ctx.Resp.Header().Add("Content-Type", "application/json")
	ctx.Resp.WriteHeader(http.StatusAccepted)
	if _, err = ctx.Resp.Write(binary); err != nil {
		log.Error("Error writing a response: %v", err)
		ctx.Error(http.StatusInternalServerError, "Error writing a response", err)
		return
	}

	accept := ap.AcceptNew(ap.IRI(fmt.Sprintf(
		"%s/follows/%d", ctx.ContextUser.APActorID(), followingID,
	)), activity)
	accept.Actor = ap.IRI(ctx.ContextUser.APActorID())
	payload, err := jsonld.WithContext(jsonld.IRI(ap.ActivityBaseURI)).Marshal(accept)
	if err != nil {
		log.Error("Unable to Marshal JSON: %v", err)
		ctx.ServerError("MarshalJSON", err)
		return
	}

	if err := pendingQueue.Push(pendingQueueItem{
		FederatedUserID: federatedUser.ID,
		Doer:            ctx.ContextUser,
		Payload:         payload,
	}); err != nil {
		log.Error("Unable to push to pending queue: %v", err)
		ctx.ServerError("pendingQueue.Push", err)
	}
}
