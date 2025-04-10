// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-FileCopyrightText: 2025 Informatyka Boguslawski sp. z o.o. sp.k. <https://www.ib.pl>
// SPDX-License-Identifier: MIT AND GPL-3.0-or-later

package middleware

import (
	"net/http"
	"strings"
)

// IsAPIPath returns true if the specified URL is an API path
func IsAPIPath(req *http.Request) bool {
	return strings.HasPrefix(req.URL.Path, "/api/")
}

// IsInternalPath returns true if the specified URL is an internal API path
func IsInternalPath(req *http.Request) bool {
	return strings.HasPrefix(req.URL.Path, "/api/internal/")
}
