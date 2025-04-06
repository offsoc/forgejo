// Copyright 2014 The Gogs Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package misc

import (
	"net/http"

	"forgejo.org/modules/markup"
	"forgejo.org/modules/markup/markdown"
	api "forgejo.org/modules/structs"
	"forgejo.org/modules/web"
	"forgejo.org/routers/common"
	"forgejo.org/services/context"
)

// Markup render markup document to HTML
func Markup(ctx *context.APIContext) {
	// swagger:operation POST /markup miscellaneous renderMarkup
	// ---
	// summary: Render a markup document as HTML
	// parameters:
	// - name: body
	//   in: body
	//   schema:
	//     "$ref": "#/definitions/MarkupOption"
	// consumes:
	// - application/json
	// produces:
	//     - text/html
	// responses:
	//   "200":
	//     "$ref": "#/responses/MarkupRender"
	//   "422":
	//     "$ref": "#/responses/validationError"

	form := web.GetForm(ctx).(*api.MarkupOption)

	if ctx.HasAPIError() {
		ctx.Error(http.StatusUnprocessableEntity, "", ctx.GetErrMsg())
		return
	}

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

// Markdown render markdown document to HTML
func Markdown(ctx *context.APIContext) {
	// swagger:operation POST /markdown miscellaneous renderMarkdown
	// ---
	// summary: Render a markdown document as HTML
	// parameters:
	// - name: body
	//   in: body
	//   schema:
	//     "$ref": "#/definitions/MarkdownOption"
	// consumes:
	// - application/json
	// produces:
	//     - text/html
	// responses:
	//   "200":
	//     "$ref": "#/responses/MarkdownRender"
	//   "422":
	//     "$ref": "#/responses/validationError"

	form := web.GetForm(ctx).(*api.MarkdownOption)

	if ctx.HasAPIError() {
		ctx.Error(http.StatusUnprocessableEntity, "", ctx.GetErrMsg())
		return
	}

	mode := "markdown"
	if form.Mode == "comment" || form.Mode == "gfm" {
		mode = form.Mode
	}

	re := common.Renderer{
		Mode:      mode,
		Text:      form.Text,
		URLPrefix: form.Context,
		IsWiki:    form.Wiki,
	}

	re.RenderMarkup(ctx.Base, ctx.Repo)
}

// MarkdownRaw render raw markdown HTML
func MarkdownRaw(ctx *context.APIContext) {
	// swagger:operation POST /markdown/raw miscellaneous renderMarkdownRaw
	// ---
	// summary: Render raw markdown as HTML
	// parameters:
	//     - name: body
	//       in: body
	//       description: Request body to render
	//       required: true
	//       schema:
	//         type: string
	// consumes:
	//     - text/plain
	// produces:
	//     - text/html
	// responses:
	//   "200":
	//     "$ref": "#/responses/MarkdownRender"
	//   "422":
	//     "$ref": "#/responses/validationError"
	defer ctx.Req.Body.Close()
	if err := markdown.RenderRaw(&markup.RenderContext{
		Ctx: ctx,
	}, ctx.Req.Body, ctx.Resp); err != nil {
		ctx.InternalServerError(err)
		return
	}
}
