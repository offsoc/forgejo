// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package federation

import (
	"fmt"
	"net/http"

	"forgejo.org/models/user"
	"forgejo.org/modules/forgefed"
	"forgejo.org/modules/log"
	context_service "forgejo.org/services/context"

	ap "github.com/go-ap/activitypub"
	"github.com/go-ap/jsonld"
)

func processPersonFollow(ctx *context_service.APIContext, activity *ap.Activity) {
	follow, err := forgefed.NewForgeFollowFromAp(*activity)
	if err != nil {
		log.Error("Invalid follow activity: %s", err)
		ctx.Error(http.StatusNotAcceptable, "Invalid follow activity", err)
		return
	}

	actorURI := follow.Actor.GetLink().String()
	_, federatedUser, federationHost, err := FindOrCreateFederatedUser(ctx.Base, actorURI)
	if err != nil {
		log.Error("Error finding or creating federated user (%s): %v", actorURI, err)
		ctx.Error(http.StatusNotAcceptable, "Federated user not found", err)
		return
	}

	following, err := user.IsFollowingAp(ctx, ctx.ContextUser, federatedUser)
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

	follower, err := user.AddFollower(ctx, ctx.ContextUser, federatedUser)
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
		"%s#accepts/follow/%d", ctx.ContextUser.APActorID(), follower.ID,
	)), follow)
	accept.Actor = ap.IRI(ctx.ContextUser.APActorID())
	payload, err := jsonld.WithContext(jsonld.IRI(ap.ActivityBaseURI)).Marshal(accept)
	if err != nil {
		log.Error("Unable to Marshal JSON: %v", err)
		ctx.ServerError("MarshalJSON", err)
		return
	}

	hostURL := federationHost.AsURL()
	if err := pendingQueue.Push(pendingQueueItem{
		InboxURL: hostURL.JoinPath(federatedUser.InboxPath).String(),
		Doer:     ctx.ContextUser,
		Payload:  payload,
	}); err != nil {
		log.Error("Unable to push to pending queue: %v", err)
		ctx.ServerError("pendingQueue.Push", err)
	}
}
