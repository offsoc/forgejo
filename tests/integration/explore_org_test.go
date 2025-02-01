// Copyright 2024 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"testing"

	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/test"
	"code.gitea.io/gitea/tests"

	"github.com/stretchr/testify/assert"
)

func TestExploreOrg(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// Set the default sort order
	defer test.MockVariableValue(&setting.UI.ExploreDefaultSort, "alphabetically")()

	cases := []struct{ sortOrder, expected string }{
		{"", "?sort=" + setting.UI.ExploreDefaultSort + "&q="},
		{"newest", "?sort=newest&q="},
		{"oldest", "?sort=oldest&q="},
		{"alphabetically", "?sort=alphabetically&q="},
		{"reversealphabetically", "?sort=reversealphabetically&q="},
	}
	for _, c := range cases {
		req := NewRequest(t, "GET", "/explore/organizations?sort="+c.sortOrder)
		resp := MakeRequest(t, req, http.StatusOK)
		h := NewHTMLParser(t, resp.Body)
		href, _ := h.Find(`.ui.dropdown .menu a.active.item[href^="?sort="]`).Attr("href")
		assert.Equal(t, c.expected, href)
	}

	// these sort orders shouldn't be supported, to avoid leaking user activity
	cases404 := []string{
		"/explore/organizations?sort=mostMembers",
		"/explore/organizations?sort=leastGroups",
		"/explore/organizations?sort=leastupdate",
		"/explore/organizations?sort=reverseleastupdate",
	}
	for _, c := range cases404 {
		req := NewRequest(t, "GET", c).SetHeader("Accept", "text/html")
		MakeRequest(t, req, http.StatusNotFound)
	}
}
