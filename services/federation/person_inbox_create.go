// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package federation

import (
	"fmt"
	"net/http"

	"code.gitea.io/gitea/models/forgefed"
	fm "code.gitea.io/gitea/modules/forgefed"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/validation"
	context_service "code.gitea.io/gitea/services/context"

	ap "github.com/go-ap/activitypub"
)

func processPersonInboxCreate(ctx *context_service.APIContext, activity *ap.Activity) {
	createAct := fm.ForgeUserActivity{Activity: *activity}

	if res, err := validation.IsValid(createAct); !res {
		log.Error("Invalid user activity: %v", activity)
		ctx.Error(http.StatusNotAcceptable, "Invalid user activity", err)
		return
	}

	if createAct.Object.GetType() != ap.NoteType {
		log.Error("Invalid object type for Create activity: %v", createAct.Object.GetType())
		ctx.Error(http.StatusNotAcceptable, "Invalid object type for Create activity", fmt.Errorf("Invalid object type for Create activity: %v", createAct.Object.GetType()))
		return
	}

	a := createAct.Object.(*ap.Object)
	userActivity := fm.ForgeUserActivityNote{Object: *a}
	act := fm.ForgeUserActivity{Activity: *activity}

	actorURI := act.Actor.GetLink().String()
	if _, _, _, err := findOrCreateFederatedUser(ctx, actorURI); err != nil {
		log.Error("Error finding or creating federated user (%s): %v", actorURI, err)
		return
	}

	if err := forgefed.AddUserActivity(ctx, ctx.ContextUser.ID, actorURI, &userActivity); err != nil {
		log.Error("Unable to record activity: %v", err)
		ctx.Error(http.StatusInternalServerError, "Unable to record activity", err)
		return
	}

	ctx.Status(http.StatusNoContent)
}
