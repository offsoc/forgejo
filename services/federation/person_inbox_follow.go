// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package federation

import (
	"fmt"
	"net/http"

	"code.gitea.io/gitea/models/forgefed"
	"code.gitea.io/gitea/modules/json"
	"code.gitea.io/gitea/modules/log"
	context_service "code.gitea.io/gitea/services/context"

	ap "github.com/go-ap/activitypub"
	"github.com/go-ap/jsonld"
)

func processPersonFollow(ctx *context_service.APIContext, activity *ap.Activity) error {
	if activity.Object.GetLink().String() != ctx.ContextUser.APActorID() {
		ctx.Error(http.StatusNotAcceptable, "Wrong user to follow", fmt.Errorf("User to follow does not match the inbox owner"))
		return nil
	}

	if activity.Actor.GetLink().String() == "" {
		ctx.Error(http.StatusNotAcceptable, "Missing actor", fmt.Errorf("Missing Actor"))
		return nil
	}

	actorURI := activity.Actor.GetLink().String()
	_, federatedUser, _, err := findOrCreateFederatedUser(ctx, actorURI)
	if err != nil {
		return nil
	}

	followingID, err := forgefed.AddFollower(ctx, ctx.ContextUser.ID, federatedUser.ID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "Unable to add follower", err)
		return err
	}

	// Respond back with an accept
	binary, err := json.Marshal(map[string]string{"status": "Accepted"})
	if err != nil {
		ctx.ServerError("MarshalJSON", err)
		return err
	}
	ctx.Resp.Header().Add("Content-Type", "application/json")
	ctx.Resp.WriteHeader(http.StatusAccepted)
	if _, err = ctx.Resp.Write(binary); err != nil {
		log.Error("write to resp err: %v", err)
		return err
	}

	accept := ap.AcceptNew(ap.IRI(fmt.Sprintf(
		"%s/follows/%d", ctx.ContextUser.APActorID(), followingID,
	)), activity)
	accept.Actor = ap.IRI(ctx.ContextUser.APActorID())
	payload, err := jsonld.WithContext(jsonld.IRI(ap.ActivityBaseURI)).Marshal(accept)
	if err != nil {
		return err
	}

	return pendingQueue.Push(pendingQueueItem{
		FederatedUserID: federatedUser.ID,
		Doer:            ctx.ContextUser,
		Payload:         payload,
	})
}
