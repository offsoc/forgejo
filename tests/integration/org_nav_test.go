// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"strings"
	"testing"

	"forgejo.org/modules/translation"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
)

// This test makes sure that organization members are able to navigate between `/<orgname>` and `/org/<orgname>/<section>` freely.
// The `/org/<orgname>/<section>` page is only accessible to the members of the organization. It doesn't have
// a special logic to show the button or not.
// The `/<orgname>` page utilizes the `IsOrganizationMember` function to show the button for navigation to
// the organization dashboard. That function is covered by a test and is supposed to be true for the
// owners/admins/members of the organization.
func TestOrgNavigationDashboard(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	locale := translation.NewLocale("en-US")

	// Login as the future organization admin and create an organization
	session1 := loginUser(t, "user2")
	session1.MakeRequest(t, NewRequestWithValues(t, "POST", "/org/create", map[string]string{
		"_csrf":                         GetCSRF(t, session1, "/org/create"),
		"org_name":                      "org_navigation_test",
		"visibility":                    "0",
		"repo_admin_change_team_access": "on",
	}), http.StatusSeeOther)

	// Check if the "Open dashboard" button is available to the org admin (member)
	resp := session1.MakeRequest(t, NewRequest(t, "GET", "/org_navigation_test"), http.StatusOK)
	doc := NewHTMLParser(t, resp.Body)
	doc.AssertElement(t, "#org-info a[href='/org/org_navigation_test/dashboard']", true)

	// Verify the "New repository" and "New migration" buttons
	links := doc.Find(".organization.profile .grid .column .center")
	assert.Equal(t, locale.TrString("new_repo.link"), strings.TrimSpace(links.Find("a[href^='/repo/create?org=']").Text()))
	assert.Equal(t, locale.TrString("new_migrate.link"), strings.TrimSpace(links.Find("a[href^='/repo/migrate?org=']").Text()))

	// Check if the "View <orgname>" button is available on dashboard for the org admin (member)
	resp = session1.MakeRequest(t, NewRequest(t, "GET", "/org/org_navigation_test/dashboard"), http.StatusOK)
	doc = NewHTMLParser(t, resp.Body)
	doc.AssertElement(t, ".dashboard .secondary-nav a[href='/org_navigation_test']", true)

	// Login a non-member user
	session2 := loginUser(t, "user4")

	// Check if the "Open dashboard" button is available to non-member
	resp = session2.MakeRequest(t, NewRequest(t, "GET", "/org_navigation_test"), http.StatusOK)
	doc = NewHTMLParser(t, resp.Body)
	doc.AssertElement(t, "#org-info a[href='/org/org_navigation_test/dashboard']", false)

	// There's no need to test "View <orgname>" button on dashboard as non-member
	// because this page is not supposed to be visitable for this user
}
