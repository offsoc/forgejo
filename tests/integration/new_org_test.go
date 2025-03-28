// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"forgejo.org/modules/translation"

	"github.com/stretchr/testify/assert"
)

func TestNewOrganizationForm(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, giteaURL *url.URL) {
		session := loginUser(t, "user1")
		locale := translation.NewLocale("en-US")

		response := session.MakeRequest(t, NewRequest(t, "GET", "/org/create"), http.StatusOK)
		page := NewHTMLParser(t, response.Body)

		// Verify page title
		title := page.Find("title").Text()
		assert.Contains(t, title, locale.TrString("new_org.title"))

		// Verify page form
		_, exists := page.Find("form[action='/org/create']").Attr("method")
		assert.True(t, exists)

		// Verify page header
		header := strings.TrimSpace(page.Find(".form[action='/org/create'] .header").Text())
		assert.Equal(t, locale.TrString("new_org.title"), header)
	})
}
