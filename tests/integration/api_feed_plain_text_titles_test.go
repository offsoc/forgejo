// Copyright 2023 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"testing"

	"forgejo.org/modules/setting"
	"forgejo.org/modules/test"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
)

func TestFeedPlainTextTitles(t *testing.T) {
	// This test verifies that items' titles in feeds are generated as plain text.
	// See https://codeberg.org/forgejo/forgejo/pulls/1595
	defer test.MockVariableValue(&setting.UI.DefaultShowFullName, true)()

	t.Run("Feed plain text titles", func(t *testing.T) {
		t.Run("Atom", func(t *testing.T) {
			defer tests.PrepareTestEnv(t)()

			req := NewRequest(t, "GET", "/user2/repo1.atom")
			resp := MakeRequest(t, req, http.StatusOK)

			data := resp.Body.String()
			assert.Contains(t, data, "<title>the_1-user.with.all.allowedChars closed issue user2/repo1#4</title>")
		})

		t.Run("RSS", func(t *testing.T) {
			defer tests.PrepareTestEnv(t)()

			req := NewRequest(t, "GET", "/user2/repo1.rss")
			resp := MakeRequest(t, req, http.StatusOK)

			data := resp.Body.String()
			assert.Contains(t, data, "<title>the_1-user.with.all.allowedChars closed issue user2/repo1#4</title>")
		})
	})
}
