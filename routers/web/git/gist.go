// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package git

import (
	"net/http"
	"strings"

	gist_model "code.gitea.io/gitea/models/gist"
	"code.gitea.io/gitea/services/context"
)

type serviceHandlerGist struct {
	gist *gist_model.Gist
}

func (h *serviceHandlerGist) Init(ctx *context.Context, isPull, receivePack bool) bool {
	gistUUID := strings.TrimSuffix(strings.ToLower(ctx.Params(":gistuuid")), ".git")

	var err error

	h.gist, err = gist_model.GetGistByUUID(ctx, gistUUID)
	if err != nil {
		if gist_model.IsErrGistNotExist(err) {
			if !ctx.IsSigned {
				ctx.Resp.Header().Set("WWW-Authenticate", `Basic realm="Gitea"`)
				ctx.Error(http.StatusUnauthorized)
				return false
			}

			ctx.NotFound("", nil)
		} else {
			ctx.ServerError("GetGistByUUID", err)
		}
		return false
	}

	if !h.gist.HasAccess(ctx.Doer) {
		if !ctx.IsSigned {
			ctx.Resp.Header().Set("WWW-Authenticate", `Basic realm="Gitea"`)
			ctx.Error(http.StatusUnauthorized)
			return false
		}

		ctx.NotFound("", nil)
		return false
	}

	if !isPull {
		if !ctx.IsSigned {
			ctx.Resp.Header().Set("WWW-Authenticate", `Basic realm="Gitea"`)
			ctx.Error(http.StatusUnauthorized)
			return false
		}

		if !h.gist.IsOwner(ctx.Doer) {
			ctx.PlainText(http.StatusForbidden, "not the owner")
		}
	}

	return true
}

func (h *serviceHandlerGist) GetRepoPath() string {
	return h.gist.GetRepoPath()
}

func (h *serviceHandlerGist) GetEnviron() []string {
	return make([]string, 0)
}
