// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"strings"
	"testing"

	"code.gitea.io/gitea/models/unittest"
	"code.gitea.io/gitea/modules/translation"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserDashboardActionLinks(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	session := loginUser(t, "user1")
	locale := translation.NewLocale("en-US")

	response := session.MakeRequest(t, NewRequest(t, "GET", "/"), http.StatusOK)
	page := NewHTMLParser(t, response.Body)
	links := page.Find("#navbar .dropdown[data-tooltip-content='Create…'] .menu")
	assert.EqualValues(t, locale.TrString("new_repo.link"), strings.TrimSpace(links.Find("a[href='/repo/create']").Text()))
	assert.EqualValues(t, locale.TrString("new_migrate.link"), strings.TrimSpace(links.Find("a[href='/repo/migrate']").Text()))
	assert.EqualValues(t, locale.TrString("new_org.link"), strings.TrimSpace(links.Find("a[href='/org/create']").Text()))
}
