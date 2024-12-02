// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package swagger

import (
	api "code.gitea.io/gitea/modules/structs"
)

// Gist
// swagger:response Gist
type swaggerResponseGist struct {
	// in:body
	Body api.Gist `json:"body"`
}

// GistList
// swagger:response GistList
type swaggerResponseGistList struct {
	// in:body
	Body api.GistList `json:"body"`
}

// GistFiles
// swagger:response GistFiles
type swaggerResponseGistFiles struct {
	// in:body
	Body []api.GistFile `json:"body"`
}
