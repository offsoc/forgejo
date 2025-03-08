// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"net/http"
	"testing"

	"code.gitea.io/gitea/tests"

	"github.com/stretchr/testify/assert"
)

func TestThemeChange(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	user := loginUser(t, "user2")

	testSelectedTheme(t, user, "forgejo-auto")

	testChangeTheme(t, user, "forgejo-dark")
	testSelectedTheme(t, user, "forgejo-dark")
}

// testSelectedTheme checks that the expected theme is used in html[data-theme]
// and is default on appearance page
func testSelectedTheme(t *testing.T, session *TestSession, expectedTheme string) {
	t.Helper()
	response := session.MakeRequest(t, NewRequest(t, "GET", "/user/settings/appearance"), http.StatusOK)
	page := NewHTMLParser(t, response.Body)

	dataTheme, dataThemeExists := page.Find("html").Attr("data-theme")
	assert.True(t, dataThemeExists)
	assert.EqualValues(t, expectedTheme, dataTheme)

	selectorTheme, selectorThemeExists := page.Find("form[action='/user/settings/appearance/theme'] input[name='theme']").Attr("value")
	assert.True(t, selectorThemeExists)
	assert.EqualValues(t, expectedTheme, selectorTheme)
}

// testSelectedTheme changes user's theme
func testChangeTheme(t *testing.T, session *TestSession, newTheme string) {
	t.Helper()
	session.MakeRequest(t, NewRequestWithValues(t, "POST", "/user/settings/appearance/theme", map[string]string{
		"_csrf": GetCSRF(t, session, "/user/settings/appearance"),
		"theme": newTheme,
	}), http.StatusSeeOther)
}
