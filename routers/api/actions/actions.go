// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package actions

import (
	"net/http"

	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/modules/web/routing"
	"code.gitea.io/gitea/routers/api/actions/ping"
	"code.gitea.io/gitea/routers/api/actions/runner"
)

func Routes(prefix string) *web.Route {
	m := web.NewRoute()

	path, handler := ping.NewPingServiceHandler()
	m.Post(path+"*", http.StripPrefix(prefix, handler).ServeHTTP)

	path, handler = runner.NewRunnerServiceHandler()
	m.Post(path+"*", routing.MarkLongPolling, http.StripPrefix(prefix, handler).ServeHTTP)

	return m
}
