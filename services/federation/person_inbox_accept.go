// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package federation

import (
	"fmt"
	"net/http"

	context_service "code.gitea.io/gitea/services/context"

	ap "github.com/go-ap/activitypub"
)

func processPersonInboxAccept(ctx *context_service.APIContext, activity *ap.Activity) {
	if activity.Object.GetType() != ap.FollowType {
		ctx.Error(http.StatusNotAcceptable, "Invalid object type for Accept activity", fmt.Errorf("Invalid object type for Accept activity: %v", activity.Object.GetType()))
		return
	}

	// We currently do not do anything here, we just drop it.

	ctx.Status(http.StatusNoContent)
}
