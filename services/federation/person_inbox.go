// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package federation

import (
	"fmt"
	"net/http"

	"forgejo.org/modules/log"
	context_service "forgejo.org/services/context"

	ap "github.com/go-ap/activitypub"
)

func ProcessPersonInbox(ctx *context_service.APIContext, form any) {
	activity := form.(*ap.Activity)

	switch activity.Type {
	case ap.CreateType:
		processPersonInboxCreate(ctx, activity)
		return
	case ap.FollowType:
		processPersonFollow(ctx, activity)
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
