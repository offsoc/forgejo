// Copyright 2014 The Gogs Authors. All rights reserved.
// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package common

import (
	"fmt"
	"net/http"
	"strings"

	"forgejo.org/modules/markup"
	"forgejo.org/modules/markup/markdown"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/util"
	"forgejo.org/services/context"

	"mvdan.cc/xurls/v2"
)

type Renderer struct {
	Mode, Text, URLPrefix, FilePath, BranchPath string
	IsWiki                                      bool
}

// RenderMarkup renders markup text for the /markup and /markdown endpoints
func (re *Renderer) RenderMarkup(ctx *context.Base, repo *context.Repository) {
	var markupType string
	relativePath := ""

	if len(re.Text) == 0 {
		_, _ = ctx.Write([]byte(""))
		return
	}

	switch re.Mode {
	case "markdown":
		// Raw markdown
		if err := markdown.RenderRaw(&markup.RenderContext{
			Ctx: ctx,
			Links: markup.Links{
				AbsolutePrefix: true,
				Base:           re.URLPrefix,
			},
		}, strings.NewReader(re.Text), ctx.Resp); err != nil {
			ctx.Error(http.StatusInternalServerError, err.Error())
		}
		return
	case "comment":
		// Comment as markdown
		markupType = markdown.MarkupName
	case "gfm":
		// Github Flavored Markdown as document
		markupType = markdown.MarkupName
	case "file":
		// File as document based on file extension
		markupType = ""
		relativePath = re.FilePath
	default:
		ctx.Error(http.StatusUnprocessableEntity, fmt.Sprintf("Unknown mode: %s", re.Mode))
		return
	}

	if !strings.HasPrefix(setting.AppSubURL+"/", re.URLPrefix) {
		// check if urlPrefix is already set to a URL
		linkRegex, _ := xurls.StrictMatchingScheme("https?://")
		m := linkRegex.FindStringIndex(re.URLPrefix)
		if m == nil {
			re.URLPrefix = util.URLJoin(setting.AppURL, re.URLPrefix)
		}
	}

	meta := map[string]string{}
	if repo != nil && repo.Repository != nil {
		if re.Mode == "comment" {
			meta = repo.Repository.ComposeMetas(ctx)
		} else {
			meta = repo.Repository.ComposeDocumentMetas(ctx)
		}
	}
	if re.Mode != "comment" {
		meta["mode"] = "document"
	}

	if err := markup.Render(&markup.RenderContext{
		Ctx: ctx,
		Links: markup.Links{
			AbsolutePrefix: true,
			Base:           re.URLPrefix,
			BranchPath:     re.BranchPath,
		},
		Metas:        meta,
		IsWiki:       re.IsWiki,
		Type:         markupType,
		RelativePath: relativePath,
	}, strings.NewReader(re.Text), ctx.Resp); err != nil {
		if markup.IsErrUnsupportedRenderExtension(err) {
			ctx.Error(http.StatusUnprocessableEntity, err.Error())
		} else {
			ctx.Error(http.StatusInternalServerError, err.Error())
		}
		return
	}
}
