// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package migrations

import (
	"io"
	"net/http"

	_ "unsafe" // Needed for go:linkname support

	gitea_sdk "code.gitea.io/sdk/gitea"
)

//go:linkname getParsedResponse code.gitea.io/sdk/gitea.(*Client).getParsedResponse
func getParsedResponse(client *gitea_sdk.Client, method, path string, header http.Header, body io.Reader, obj any) (*gitea_sdk.Response, error)
