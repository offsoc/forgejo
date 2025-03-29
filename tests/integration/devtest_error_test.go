// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"net/http"
	"testing"

	"forgejo.org/modules/setting"
	"forgejo.org/modules/test"
	"forgejo.org/routers"

	"github.com/stretchr/testify/assert"
)

// `/devtest/error/{errcode}` provides a convenient way of testing various
// error pages sometimes which can be hard to reach otherwise.
// This file is a test of various attributes on those pages.

func TestDevtestErrorpages(t *testing.T) {
	defer test.MockVariableValue(&setting.IsProd, false)()
	defer test.MockVariableValue(&testWebRoutes, routers.NormalRoutes())()

	t.Run("Server error", func(t *testing.T) {
		// `/devtest/error/x` returns 500 for any x by default.
		// `/500` is simply for good look here
		req := NewRequest(t, "GET", "/devtest/error/500")
		resp := MakeRequest(t, req, http.StatusInternalServerError)
		doc := NewHTMLParser(t, resp.Body)
		assert.Equal(t, "500", doc.Find(".error-code").Text())
		assert.Contains(t, doc.Find("head title").Text(), "Internal server error")
	})

	t.Run("Page not found",
		func(t *testing.T) {
			req := NewRequest(t, "GET", "/devtest/error/404").
				// Without this header `notFoundInternal` returns plaintext error message
				SetHeader("Accept", "text/html")
			resp := MakeRequest(t, req, http.StatusNotFound)
			doc := NewHTMLParser(t, resp.Body)
			assert.Equal(t, "404", doc.Find(".error-code").Text())
			assert.Contains(t, doc.Find("head title").Text(), "Page not found")
		})

	t.Run("Quota exhaustion",
		func(t *testing.T) {
			req := NewRequest(t, "GET", "/devtest/error/413")
			resp := MakeRequest(t, req, http.StatusRequestEntityTooLarge)
			doc := NewHTMLParser(t, resp.Body)
			assert.Equal(t, "413", doc.Find(".error-code").Text())
		})
}
