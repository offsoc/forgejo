// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT
package common

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"forgejo.org/modules/web"

	chi "github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

func TestStripSlashesMiddleware(t *testing.T) {
	type test struct {
		name               string
		expectedPath       string
		expectedNormalPath string
		inputPath          string
	}

	tests := []test{
		{
			name:         "path with multiple slashes",
			inputPath:    "https://github.com///go-gitea//gitea.git",
			expectedPath: "/go-gitea/gitea.git",
		},
		{
			name:         "path with no slashes",
			inputPath:    "https://github.com/go-gitea/gitea.git",
			expectedPath: "/go-gitea/gitea.git",
		},
		{
			name:         "path with slashes in the middle",
			inputPath:    "https://git.data.coop//halfd/new-website.git",
			expectedPath: "/halfd/new-website.git",
		},
		{
			name:         "path with slashes in the middle",
			inputPath:    "https://git.data.coop//halfd/new-website.git",
			expectedPath: "/halfd/new-website.git",
		},
		{
			name:         "path with slashes in the end",
			inputPath:    "/user2//repo1/",
			expectedPath: "/user2/repo1",
		},
		{
			name:         "path with slashes in the beginning",
			inputPath:    "https://codeberg.org//user2/repo1/",
			expectedPath: "/user2/repo1",
		},
		{
			name:         "path with slashes and query params",
			inputPath:    "/repo//migrate?service_type=3",
			expectedPath: "/repo/migrate",
		},
		{
			name:               "path with encoded slash",
			inputPath:          "/user2/%2F%2Frepo1",
			expectedPath:       "/user2/%2F%2Frepo1",
			expectedNormalPath: "/user2/repo1",
		},
		{
			name:               "path with space",
			inputPath:          "/assets/css/theme%20cappuccino.css",
			expectedPath:       "/assets/css/theme%20cappuccino.css",
			expectedNormalPath: "/assets/css/theme cappuccino.css",
		},
	}

	for _, tt := range tests {
		r := web.NewRoute()
		r.Use(stripSlashesMiddleware)

		called := false
		r.Get("*", func(w http.ResponseWriter, r *http.Request) {
			if tt.expectedNormalPath != "" {
				assert.Equal(t, tt.expectedNormalPath, r.URL.Path)
			} else {
				assert.Equal(t, tt.expectedPath, r.URL.Path)
			}

			rctx := chi.RouteContext(r.Context())
			assert.Equal(t, tt.expectedPath, rctx.RoutePath)

			called = true
		})

		// create a mock request to use
		req := httptest.NewRequest("GET", tt.inputPath, nil)
		r.ServeHTTP(httptest.NewRecorder(), req)
		assert.True(t, called)
	}
}
