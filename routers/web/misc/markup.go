// Copyright 2014 The Gogs Authors. All rights reserved.
// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package misc

import (
	api "code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/routers/common"
	"code.gitea.io/gitea/services/context"
)

// Markup render markup document to HTML
func Markup(ctx *context.Context) {
	form := web.GetForm(ctx).(*api.MarkupOption)

	re := common.Renderer{
		Mode:       form.Mode,
		Text:       form.Text,
		URLPrefix:  form.Context,
		FilePath:   form.FilePath,
		BranchPath: form.BranchPath,
		IsWiki:     form.Wiki,
	}

	re.RenderMarkup(ctx.Base, ctx.Repo)
}
