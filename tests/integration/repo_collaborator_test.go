// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"strings"
	"testing"

	"code.gitea.io/gitea/tests"

	"github.com/stretchr/testify/assert"
)

// TestRepoCollaborators is a test for contents of Collaborators tab in the repo settings
// It only covers a few elements and can be extended as needed
func TestRepoCollaborators(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	session := loginUser(t, "user2")

	// Visit Collaborators tab of repo settings
	response := session.MakeRequest(t, NewRequest(t, "GET", "/user2/repo1/settings/collaboration"), http.StatusOK)
	page := NewHTMLParser(t, response.Body).Find(".repo-setting-content")

	// Veirfy header
	assert.EqualValues(t, "Collaborators", strings.TrimSpace(page.Find("h4").Text()))

	// Veirfy button text
	page = page.Find("#repo-collab-form")
	assert.EqualValues(t, "Add collaborator", strings.TrimSpace(page.Find("button.primary").Text()))

	// Veirfy placeholder
	placeholder, exists := page.Find("#search-user-box input").Attr("placeholder")
	assert.True(t, exists)
	assert.EqualValues(t, "Search users…", placeholder)
}
