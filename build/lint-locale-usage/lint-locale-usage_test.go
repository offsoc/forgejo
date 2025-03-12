// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package main

import (
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func HandleGoFileWrapped(t *testing.T, fname, src string) []string {
	var ret []string
	omh := OnMsgidHandler(func(fset *token.FileSet, pos token.Pos, msgid string) {
		ret = append(ret, msgid)
	})
	require.NoError(t, omh.HandleGoFile(fname, src))
	return ret
}

func HandleTemplateFileWrapped(t *testing.T, fname, src string) []string {
	var ret []string
	omh := OnMsgidHandler(func(fset *token.FileSet, pos token.Pos, msgid string) {
		ret = append(ret, msgid)
	})
	require.NoError(t, omh.HandleTemplateFile(fname, src))
	return ret
}

func TestUsagesParser(t *testing.T) {
	t.Run("go, simple", func(t *testing.T) {
		assert.EqualValues(t,
			[]string{"what.an.example"},
			HandleGoFileWrapped(t, "<g1>", "package main\nfunc Render(ctx *context.Context) string { return ctx.Tr(\"what.an.example\"); }\n"))
	})

	t.Run("template, simple", func(t *testing.T) {
		assert.EqualValues(t,
			[]string{"what.an.example"},
			HandleTemplateFileWrapped(t, "<t1>", "{{ ctx.Locale.Tr \"what.an.example\" }}\n"))
	})
}
