// Copyright 2024-2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"forgejo.org/models/unittest"
	"forgejo.org/modules/translation"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This test verifies common elements that are visible on all pages but most
// likely to be first seen on `/`
func TestCommonNavigationElements(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	session := loginUser(t, "user1")
	locale := translation.NewLocale("en-US")

	response := session.MakeRequest(t, NewRequest(t, "GET", "/"), http.StatusOK)
	page := NewHTMLParser(t, response.Body)

	// Navbar
	links := page.Find("#navbar .dropdown[data-tooltip-content='Create…'] .menu")
	assert.Equal(t, locale.TrString("new_repo.link"), strings.TrimSpace(links.Find("a[href='/repo/create']").Text()))
	assert.Equal(t, locale.TrString("new_migrate.link"), strings.TrimSpace(links.Find("a[href='/repo/migrate']").Text()))
	assert.Equal(t, locale.TrString("new_org.link"), strings.TrimSpace(links.Find("a[href='/org/create']").Text()))

	// After footer: index.js
	page.AssertElement(t, "script[src^='/assets/js/index.js']", true)
	onerror, _ := page.Find("script[src^='/assets/js/index.js']").Attr("onerror")
	expected := fmt.Sprintf("alert('%s'.replace('{path}', this.src))", locale.TrString("alert.asset_load_failed"))
	assert.Equal(t, expected, onerror)
}
