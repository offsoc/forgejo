// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"net/http"
	"strconv"
	"testing"

	"forgejo.org/modules/setting"
	"forgejo.org/modules/structs"
	"forgejo.org/modules/test"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
)

// TestUserProfileAttributes ensures visibility and correctness of elements related to activity of a user:
// - RSS/atom feed links (doesn't test `other.ENABLE_FEED:false`) and a few other links nearby
// - Public activity tab
// - Banner/hint in the tab
// - "Configure" link in the hint
// These elements might depend on the following:
// - Profile visibility
// - Public activity visibility
func TestUserProfileAttributes(t *testing.T) {
	defer test.MockVariableValue(&setting.AppSubURL, "/sub")()
	defer tests.PrepareTestEnv(t)()
	// This test needs multiple users with different access statuses to check for all possible states
	userAdmin := loginUser(t, "user1")
	userRegular := loginUser(t, "user2")
	// Activity availability should be the same for guest and another non-admin user, so this is not tested separately
	userGuest := emptyTestSession(t)

	// = Public profile, public activity =

	// Set activity visibility of user2 to public. This is the default, but won't hurt to set it before testing.
	testChangeUserActivityVisibility(t, userRegular, "off")

	// Verify availability of activity tab and other links
	testUser2ActivityLinksAvailability(t, userAdmin, true, true, false)
	testUser2ActivityLinksAvailability(t, userRegular, true, false, true)
	testUser2ActivityLinksAvailability(t, userGuest, true, false, false)

	// Verify the hint for all types of users: admin, self, guest
	testUser2ActivityVisibility(t, userAdmin, "This activity is visible to everyone, but as an administrator you can also see interactions in private spaces.", true)
	testUser2ActivityVisibility(t, userRegular, "Your activity is visible to everyone, except for interactions in private spaces. Configure.", true)
	testUser2ActivityVisibility(t, userGuest, "", true)

	// = Private profile, but public activity =

	// Set profile visibility of user2 to private
	testChangeUserProfileVisibility(t, userRegular, structs.VisibleTypePrivate)

	// When profile activity is configured as public, but the profile is private, tell the user about this and link to visibility settings.
	hintLink := testUser2ActivityVisibility(t, userRegular, "Your activity is only visible to you and the instance administrators because your profile is private. Configure.", true)
	assert.Equal(t, "/sub/user/settings#visibility-setting", hintLink)

	// When the profile is private, tell the admin about this.
	testUser2ActivityVisibility(t, userAdmin, "This activity is visible to you because you're an administrator, but the user wants it to remain private.", true)

	// Set profile visibility of user2 back to public
	testChangeUserProfileVisibility(t, userRegular, structs.VisibleTypePublic)

	// = Private activity =

	// Set activity visibility of user2 to private
	testChangeUserActivityVisibility(t, userRegular, "on")

	// Verify availability of activity tab and other links
	testUser2ActivityLinksAvailability(t, userAdmin, true, true, false)
	testUser2ActivityLinksAvailability(t, userRegular, true, false, true)
	testUser2ActivityLinksAvailability(t, userGuest, false, false, false)

	// Verify the hint for all types of users: admin, self, guest
	testUser2ActivityVisibility(t, userAdmin, "This activity is visible to you because you're an administrator, but the user wants it to remain private.", true)
	hintLink = testUser2ActivityVisibility(t, userRegular, "Your activity is only visible to you and the instance administrators. Configure.", true)
	testUser2ActivityVisibility(t, userGuest, "This user has disabled the public visibility of the activity.", false)

	// Verify that Configure link is correct
	assert.Equal(t, "/sub/user/settings#keep-activity-private", hintLink)
}

// testChangeUserActivityVisibility allows to easily change visibility of public activity for a user
func testChangeUserActivityVisibility(t *testing.T, session *TestSession, newState string) {
	t.Helper()
	session.MakeRequest(t, NewRequestWithValues(t, "POST", "/user/settings",
		map[string]string{
			"_csrf":                 GetCSRF(t, session, "/user/settings"),
			"keep_activity_private": newState,
		}), http.StatusSeeOther)
}

// testChangeUserProfileVisibility allows to easily change visibility of user's profile
func testChangeUserProfileVisibility(t *testing.T, session *TestSession, newValue structs.VisibleType) {
	t.Helper()
	session.MakeRequest(t, NewRequestWithValues(t, "POST", "/user/settings", map[string]string{
		"_csrf":      GetCSRF(t, session, "/user/settings"),
		"visibility": strconv.Itoa(int(newValue)),
	}), http.StatusSeeOther)
}

// testUser2ActivityVisibility checks visibility of UI elements on /<user>?tab=activity
// It also returns the account visibility link if it is present on the page.
func testUser2ActivityVisibility(t *testing.T, session *TestSession, hint string, availability bool) string {
	t.Helper()
	response := session.MakeRequest(t, NewRequest(t, "GET", "/user2?tab=activity"), http.StatusOK)
	page := NewHTMLParser(t, response.Body)
	// Check hint visibility and correctness
	testSelectorEquals(t, page, "#visibility-hint", hint)
	hintLink, hintLinkExists := page.Find("#visibility-hint a").Attr("href")

	// Check that the hint aligns with the actual feed availability
	page.AssertElement(t, "#activity-feed", availability)

	// Check that the current tab is displayed and is active regardless of it's actual availability
	// For example, on /<user> it wouldn't be available to guest, but it should be still present on /<user>?tab=activity
	assert.Positive(t, page.Find("overflow-menu .active.item[href='/sub/user2?tab=activity']").Length())
	if hintLinkExists {
		return hintLink
	}
	return ""
}

// testUser2ActivityLinksAvailability checks visibility of:
// * Public activity tab on main profile page
// * user details, profile edit, feed links
func testUser2ActivityLinksAvailability(t *testing.T, session *TestSession, activity, adminLink, editLink bool) {
	t.Helper()
	response := session.MakeRequest(t, NewRequest(t, "GET", "/user2"), http.StatusOK)
	page := NewHTMLParser(t, response.Body)
	page.AssertElement(t, "overflow-menu .item[href='/sub/user2?tab=activity']", activity)

	// User details - for admins only
	page.AssertElement(t, "#profile-avatar-card a[href='/sub/admin/users/2']", adminLink)
	// Edit profile - for self only
	page.AssertElement(t, "#profile-avatar-card a[href='/sub/user/settings']", editLink)

	// Feed links
	page.AssertElement(t, "#profile-avatar-card a[href='/sub/user2.rss']", activity)
	page.AssertElement(t, "#profile-avatar-card a[href='/sub/user2.atom']", activity)
}
