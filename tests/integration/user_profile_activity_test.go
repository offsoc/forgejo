// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestUserProfileActivity ensures visibility and correctness of elements related to activity of a user:
// - RSS feed button (doesn't test `other.ENABLE_FEED:false`)
// - Public activity tab
// - Banner/hint in the tab
// - "Configure" link in the hint
func TestUserProfileActivity(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, giteaURL *url.URL) {
		// This test needs multiple users with different access statuses to check for all possible states
		userAdmin := loginUser(t, "user1")
		userRegular := loginUser(t, "user2")
		// Activity availability should be the same for guest and another non-admin user, so this is not tested separately
		userGuest := emptyTestSession(t)

		// = Public =

		// Set activity visibility of user2 to public. This is the default, but won't hurt to set it before testing.
		testChangeUserActivityVisibility(t, userRegular, "off")

		// Verify availability of RSS button and activity tab
		testUser2ActivityButtonsAvailability(t, userAdmin, true)
		testUser2ActivityButtonsAvailability(t, userRegular, true)
		testUser2ActivityButtonsAvailability(t, userGuest, true)

		// Verify the hint for all types of users: admin, self, guest
		testUser2ActivityVisibility(t, userAdmin, "This activity is visible to everyone, but as an administrator you can also see interactions in private spaces.", true)
		hintLink := testUser2ActivityVisibility(t, userRegular, "Your activity is visible to everyone, except for interactions in private spaces. Configure.", true)
		testUser2ActivityVisibility(t, userGuest, "", true)

		// When viewing own profile, the user is offered to configure activity visibility. Verify that the link is correct and works, also check that it links back to the activity tab.
		linkCorrect := assert.EqualValues(t, "/user/settings#keep-activity-private", hintLink)
		if linkCorrect {
			page := NewHTMLParser(t, userRegular.MakeRequest(t, NewRequest(t, "GET", hintLink), http.StatusOK).Body)
			activityLink, exists := page.Find(".field:has(.checkbox#keep-activity-private) .help a").Attr("href")
			assert.True(t, exists)
			assert.EqualValues(t, "/user2?tab=activity", activityLink)
		}

		// = Private =

		// Set activity visibility of user2 to private
		testChangeUserActivityVisibility(t, userRegular, "on")

		// Verify availability of RSS button and activity tab
		testUser2ActivityButtonsAvailability(t, userAdmin, true)
		testUser2ActivityButtonsAvailability(t, userRegular, true)
		testUser2ActivityButtonsAvailability(t, userGuest, false)

		// Verify the hint for all types of users: admin, self, guest
		testUser2ActivityVisibility(t, userAdmin, "This activity is visible to you because you're an administrator, but the user wants it to remain private.", true)
		hintLink = testUser2ActivityVisibility(t, userRegular, "Your activity is only visible to you and the instance administrators. Configure.", true)
		testUser2ActivityVisibility(t, userGuest, "This user has disabled the public visibility of the activity.", false)

		// Verify that Configure link is correct
		assert.EqualValues(t, "/user/settings#keep-activity-private", hintLink)
	})
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
	assert.EqualValues(t, availability, page.Find("#activity-feed").Length() > 0)

	// Check availability of RSS feed button too
	assert.EqualValues(t, availability, page.Find("#profile-avatar-card a[href='/user2.rss']").Length() > 0)

	// Check that the current tab is displayed and is active regardless of it's actual availability
	// For example, on /<user> it wouldn't be available to guest, but it should be still present on /<user>?tab=activity
	assert.True(t, page.Find("overflow-menu .active.item[href='/user2?tab=activity']").Length() > 0)

	if hintLinkExists {
		return hintLink
	}
	return ""
}

// testUser2ActivityButtonsAvailability checks visibility of Public activity tab on main profile page
func testUser2ActivityButtonsAvailability(t *testing.T, session *TestSession, buttons bool) {
	t.Helper()
	response := session.MakeRequest(t, NewRequest(t, "GET", "/user2"), http.StatusOK)
	page := NewHTMLParser(t, response.Body)
	assert.EqualValues(t, buttons, page.Find("overflow-menu .item[href='/user2?tab=activity']").Length() > 0)
}
