// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package federation

import (
	"fmt"
	"net/http"

	"forgejo.org/models/user"
	fm "forgejo.org/modules/forgefed"
	"forgejo.org/modules/log"
	context_service "forgejo.org/services/context"

	ap "github.com/go-ap/activitypub"
	"github.com/go-ap/jsonld"
)

func ProcessPersonInbox(ctx *context_service.APIContext, form any) {
	activity := form.(*ap.Activity)

	switch activity.Type {
	case ap.CreateType:
		processPersonInboxCreate(ctx, activity)
		return
	case ap.FollowType:
		processPersonFollow(ctx, form.(*fm.ForgeFollow))
		return
	case ap.UndoType:
		processPersonInboxUndo(ctx, activity)
		return
	case ap.AcceptType:
		processPersonInboxAccept(ctx, activity)
		return
	}

	log.Error("Unsupported PersonInbox activity: %v", activity.Type)
	ctx.Error(http.StatusNotAcceptable, "Unsupported acvitiy", fmt.Errorf("Unsupported activity: %v", activity.Type))
}

func FollowRemoteActor(ctx *context_service.APIContext, localUser *user.User, actorURI string) error {
	_, federatedUser, federationHost, err := FindOrCreateFederatedUser(ctx, actorURI)
	if err != nil {
		return err
	}

	followReq, err := fm.NewForgeFollow(localUser.APActorID(), actorURI)
	if err != nil {
		return err
	}

	payload, err := jsonld.WithContext(jsonld.IRI(ap.ActivityBaseURI)).
		Marshal(followReq)
	if err != nil {
		return err
	}

	hostURL := federationHost.AsURL()
	return pendingQueue.Push(pendingQueueItem{
		InboxURL: hostURL.JoinPath(federatedUser.InboxPath).String(),
		Doer:     localUser,
		Payload:  payload,
	})
}
