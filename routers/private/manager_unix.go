// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package private

import (
	"net/http"

	"forgejo.org/modules/graceful"
	"forgejo.org/services/context"
)

// Restart causes the server to perform a graceful restart
func Restart(ctx *context.PrivateContext) {
	graceful.GetManager().DoGracefulRestart()
	ctx.PlainText(http.StatusOK, "success")
}

// Shutdown causes the server to perform a graceful shutdown
func Shutdown(ctx *context.PrivateContext) {
	graceful.GetManager().DoGracefulShutdown()
	ctx.PlainText(http.StatusOK, "success")
}
