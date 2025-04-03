// Copyright 2022 The Gitea Authors. All rights reserved.
// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package activitypub

import (
	"net/http"

	user_model "forgejo.org/models/user"
	"forgejo.org/modules/activitypub"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
	"forgejo.org/services/context"

	ap "github.com/go-ap/activitypub"
	"github.com/go-ap/jsonld"
)

// Actor function returns the instance's Actor
func Actor(ctx *context.APIContext) {
	// swagger:operation GET /activitypub/actor activitypub activitypubInstanceActor
	// ---
	// summary: Returns the instance's Actor
	// produces:
	// - application/json
	// responses:
	//   "200":
	//     "$ref": "#/responses/ActivityPub"

	link := user_model.APServerActorID()
	actor := ap.ActorNew(ap.IRI(link), ap.ApplicationType)

	actor.PreferredUsername = ap.NaturalLanguageValuesNew()
	err := actor.PreferredUsername.Set("en", ap.Content(setting.Domain))
	if err != nil {
		ctx.ServerError("PreferredUsername.Set", err)
		return
	}

	actor.URL = ap.IRI(setting.AppURL)

	actor.Inbox = ap.IRI(link + "/inbox")
	actor.Outbox = ap.IRI(link + "/outbox")

	actor.PublicKey.ID = ap.IRI(link + "#main-key")
	actor.PublicKey.Owner = ap.IRI(link)

	publicKeyPem, err := activitypub.GetPublicKey(ctx, user_model.NewAPServerActor())
	if err != nil {
		ctx.ServerError("GetPublicKey", err)
		return
	}
	actor.PublicKey.PublicKeyPem = publicKeyPem

	binary, err := jsonld.WithContext(
		jsonld.IRI(ap.ActivityBaseURI),
		jsonld.IRI(ap.SecurityContextURI),
	).Marshal(actor)
	if err != nil {
		ctx.ServerError("MarshalJSON", err)
		return
	}
	ctx.Resp.Header().Add("Content-Type", activitypub.ActivityStreamsContentType)
	ctx.Resp.WriteHeader(http.StatusOK)
	if _, err = ctx.Resp.Write(binary); err != nil {
		log.Error("write to resp err: %v", err)
	}
}

// ActorInbox function handles the incoming data for the instance Actor
func ActorInbox(ctx *context.APIContext) {
	// swagger:operation POST /activitypub/actor/inbox activitypub activitypubInstanceActorInbox
	// ---
	// summary: Send to the inbox
	// produces:
	// - application/json
	// responses:
	//   "204":
	//     "$ref": "#/responses/empty"

	ctx.Status(http.StatusNoContent)
}
