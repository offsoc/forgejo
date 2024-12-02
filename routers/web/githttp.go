// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package web

import (
	"net/http"

	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/routers/web/git"
	"code.gitea.io/gitea/services/context"
)

func requireSignIn(ctx *context.Context) {
	if !setting.Service.RequireSignInView {
		return
	}

	// rely on the results of Contexter
	if !ctx.IsSigned {
		// TODO: support digit auth - which would be Authorization header with digit
		ctx.Resp.Header().Set("WWW-Authenticate", `Basic realm="Gitea"`)
		ctx.Error(http.StatusUnauthorized)
	}
}

func gitHTTPRouters(m *web.Route) {
	m.Group("", func() {
		m.Methods("POST,OPTIONS", "/git-upload-pack", git.ServiceUploadPack)
		m.Methods("POST,OPTIONS", "/git-receive-pack", git.ServiceReceivePack)
		m.Methods("GET,OPTIONS", "/info/refs", git.GetInfoRefs)
		m.Methods("GET,OPTIONS", "/HEAD", git.GetTextFile("HEAD"))
		m.Methods("GET,OPTIONS", "/objects/info/alternates", git.GetTextFile("objects/info/alternates"))
		m.Methods("GET,OPTIONS", "/objects/info/http-alternates", git.GetTextFile("objects/info/http-alternates"))
		m.Methods("GET,OPTIONS", "/objects/info/packs", git.GetInfoPacks)
		m.Methods("GET,OPTIONS", "/objects/info/{file:[^/]*}", git.GetTextFile(""))
		m.Methods("GET,OPTIONS", "/objects/{head:[0-9a-f]{2}}/{hash:[0-9a-f]{38,62}}", git.GetLooseObject)
		m.Methods("GET,OPTIONS", "/objects/pack/pack-{file:[0-9a-f]{40,64}}.pack", git.GetPackFile)
		m.Methods("GET,OPTIONS", "/objects/pack/pack-{file:[0-9a-f]{40,64}}.idx", git.GetIdxFile)
	}, ignSignInAndCsrf, requireSignIn, git.HTTPGitEnabledHandler, git.CorsHandler())
}
