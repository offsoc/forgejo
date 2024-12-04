// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package gist_test

import (
	"testing"

	api "code.gitea.io/gitea/modules/structs"
	gist_service "code.gitea.io/gitea/services/gist"

	"github.com/stretchr/testify/assert"
)

func TestGistFilesContains(t *testing.T) {
	files := make(gist_service.GistFiles, 2)

	files[0] = &api.GistFile{Name: "a.txt"}
	files[1] = &api.GistFile{Name: "b.txt"}

	assert.True(t, files.Contains("A.txt"))
	assert.False(t, files.Contains("C.txt"))
}
