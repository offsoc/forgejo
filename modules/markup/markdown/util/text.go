// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package util

import (
	"bytes"

	"github.com/yuin/goldmark/ast"
)

func textOfChildren(n ast.Node, src []byte, b *bytes.Buffer) {
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if t, ok := c.(*ast.Text); ok {
			b.Write(t.Value(src))
		} else {
			textOfChildren(c, src, b)
		}
	}
}

func Text(n ast.Node, src []byte) []byte {
	var b bytes.Buffer
	textOfChildren(n, src, &b)
	return b.Bytes()
}
