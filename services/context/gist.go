// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package context

import (
	"fmt"

	gist_model "code.gitea.io/gitea/models/gist"
)

// GistAssignment handels Context.Gist assignment
func GistAssignment(ctx *Context) {
	gistUUID := ctx.Params(":gistuuid")

	gist, err := gist_model.GetGistByUUID(ctx, gistUUID)
	if err != nil {
		if gist_model.IsErrGistNotExist(err) {
			ctx.NotFound(fmt.Sprintf("gist %s was not found", gistUUID), nil)
		} else {
			ctx.ServerError("GetGistByUUID", err)
		}
		return
	}

	if !gist.HasAccess(ctx.Doer) {
		ctx.NotFound(fmt.Sprintf("gist %s is private", gistUUID), nil)
		return
	}

	ctx.Gist = gist
}

// RequireGistowner checks if teh Doer is the Owner of the Gist
func RequireGistOwner(ctx *Context) {
	if !ctx.Gist.IsOwner(ctx.Doer) {
		ctx.NotFound(ctx.Req.URL.RequestURI(), nil)
	}
}
