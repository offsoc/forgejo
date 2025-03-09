// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"net/http"
	"strings"
	"testing"

	"code.gitea.io/gitea/tests"

	"github.com/stretchr/testify/assert"
)

func TestThemeChange(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	user := loginUser(t, "user2")

	// Verify default theme
	testSelectedTheme(t, user, "forgejo-auto", "Forgejo (follow system theme)")

	// Change theme to forgejo-dark and verify it works fine
	testChangeTheme(t, user, "forgejo-dark")
	testSelectedTheme(t, user, "forgejo-dark", "Forgejo dark")

	// Change theme to gitea-dark and also verify that it's name is not translated
	testChangeTheme(t, user, "gitea-dark")
	testSelectedTheme(t, user, "gitea-dark", "gitea-dark")
}

// testSelectedTheme checks that the expected theme is used in html[data-theme]
// and is default on appearance page
func testSelectedTheme(t *testing.T, session *TestSession, expectedTheme, expectedName string) {
	t.Helper()
	response := session.MakeRequest(t, NewRequest(t, "GET", "/user/settings/appearance"), http.StatusOK)
	page := NewHTMLParser(t, response.Body)

	dataTheme, dataThemeExists := page.Find("html").Attr("data-theme")
	assert.True(t, dataThemeExists)
	assert.EqualValues(t, expectedTheme, dataTheme)

	selectedTheme := page.Find("form[action='/user/settings/appearance/theme'] .menu .item.selected")
	selectorTheme, selectorThemeExists := selectedTheme.Attr("data-value")
	assert.True(t, selectorThemeExists)
	assert.EqualValues(t, expectedTheme, selectorTheme)
	assert.EqualValues(t, expectedName, strings.TrimSpace(selectedTheme.Text()))
}

// testSelectedTheme changes user's theme
func testChangeTheme(t *testing.T, session *TestSession, newTheme string) {
	t.Helper()
	session.MakeRequest(t, NewRequestWithValues(t, "POST", "/user/settings/appearance/theme", map[string]string{
		"_csrf": GetCSRF(t, session, "/user/settings/appearance"),
		"theme": newTheme,
	}), http.StatusSeeOther)
}
