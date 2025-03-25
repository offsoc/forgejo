// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package fuzz

import (
	"bytes"
	"context"
	"io"
	"testing"

	"forgejo.org/modules/markup"
	"forgejo.org/modules/markup/markdown"
	"forgejo.org/modules/setting"
)

var renderContext = markup.RenderContext{
	Ctx: context.Background(),
	Links: markup.Links{
		Base: "https://example.com/go-gitea/gitea",
	},
	Metas: map[string]string{
		"user": "go-gitea",
		"repo": "gitea",
	},
}

func FuzzMarkdownRenderRaw(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		setting.IsInTesting = true
		setting.AppURL = "http://localhost:3000/"
		markdown.RenderRaw(&renderContext, bytes.NewReader(data), io.Discard)
	})
}

func FuzzMarkupPostProcess(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		setting.IsInTesting = true
		setting.AppURL = "http://localhost:3000/"
		markup.PostProcess(&renderContext, bytes.NewReader(data), io.Discard)
	})
}
