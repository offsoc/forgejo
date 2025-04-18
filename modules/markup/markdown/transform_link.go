// Copyright 2024 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package markdown

import (
	"bytes"
	"slices"

	"forgejo.org/modules/markup"
	"forgejo.org/modules/setting"
	giteautil "forgejo.org/modules/util"

	"github.com/yuin/goldmark/ast"
)

func (g *ASTTransformer) transformLink(ctx *markup.RenderContext, v *ast.Link) {
	// Links need their href to munged to be a real value
	link := v.Destination

	// Do not process the link if it's not a link, starts with an hashtag
	// (indicating it's an anchor link), starts with `mailto:` or any of the
	// custom markdown URLs.
	processLink := len(link) > 0 && !markup.IsLink(link) &&
		link[0] != '#' && !bytes.HasPrefix(link, byteMailto) &&
		!slices.ContainsFunc(setting.Markdown.CustomURLSchemes, func(s string) bool {
			return bytes.HasPrefix(link, []byte(s+":"))
		})

	if processLink {
		var base string
		if ctx.IsWiki {
			base = ctx.Links.WikiLink()
		} else if ctx.Links.HasBranchInfo() {
			base = ctx.Links.SrcLink()
		} else {
			base = ctx.Links.Base
		}

		link = []byte(giteautil.URLJoin(base, string(link)))
	}
	if len(link) > 0 && link[0] == '#' {
		link = []byte("#user-content-" + string(link)[1:])
	}
	v.Destination = link
}
