// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package gist

import (
	"strings"

	"code.gitea.io/gitea/modules/highlight"
	api "code.gitea.io/gitea/modules/structs"
)

type GistFiles []*api.GistFile //revive:disable-line:exported

func (files GistFiles) Contains(name string) bool {
	for _, currentFile := range files {
		if strings.EqualFold(name, currentFile.Name) {
			return true
		}
	}

	return false
}

func (files GistFiles) Highlight() error {
	var err error

	for _, currentFile := range files {
		currentFile.HighlightedContent, _, err = highlight.File(currentFile.Name, "", []byte(currentFile.Content))
		if err != nil {
			return err
		}
	}

	return nil
}
