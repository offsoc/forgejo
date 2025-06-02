// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
)

// TestUserProfileFollows is a test for user following counters, pages and titles.
// It tests that:
// - Followers and Following tabs always have titles present and always use correct plurals
// - Followers and Following lists have correct amounts of items
// - %d followers and %following counters are always present and always have correct numbers and use correct plurals
func TestUserProfileFollows(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// This test needs 3 users to check for all possible states
	// The accounts of user3 and user4 are not functioning
	user1 := loginUser(t, "user1")
	user2 := loginUser(t, "user2")
	user5 := loginUser(t, "user5")

	followersLink := "#profile-avatar-card a[href='/user1?tab=followers']"
	followingLink := "#profile-avatar-card a[href='/user1?tab=following']"
	listHeader := ".user-cards h2"
	listItems := ".user-cards .list"
	listMsg := ".user-cards > div"

	// = No follows =

	var followCount int

	// Request the profile of user1, the Followers tab
	response := user1.MakeRequest(t, NewRequest(t, "GET", "/user1?tab=followers"), http.StatusOK)
	page := NewHTMLParser(t, response.Body)

	// Verify that user1 has no followers
	testSelectorEquals(t, page, followersLink, "0 followers")
	testSelectorEquals(t, page, listHeader, "Followers")
	page.AssertElement(t, listItems, false)
	page.AssertElement(t, listMsg, true)

	// Request the profile of user1, the Following tab
	response = user1.MakeRequest(t, NewRequest(t, "GET", "/user1?tab=following"), http.StatusOK)
	page = NewHTMLParser(t, response.Body)

	// Verify that user1 does not follow anyone
	testSelectorEquals(t, page, followingLink, "0 following")
	testSelectorEquals(t, page, listHeader, "Following")
	page.AssertElement(t, listItems, false)
	page.AssertElement(t, listMsg, true)

	// Make user1 and user2 follow each other
	testUserFollowUser(t, user1, "user2")
	testUserFollowUser(t, user2, "user1")

	// = 1 follow each =

	followCount++

	// Request the profile of user1, the Followers tab
	response = user1.MakeRequest(t, NewRequest(t, "GET", "/user1?tab=followers"), http.StatusOK)
	page = NewHTMLParser(t, response.Body)

	// Verify it is now followed by 1 user
	testSelectorEquals(t, page, followersLink, "1 follower")
	testSelectorEquals(t, page, listHeader, "Follower")
	testListCount(t, page, listItems, followCount)
	page.AssertElement(t, listMsg, false)

	// Request the profile of user1, the Following tab
	response = user1.MakeRequest(t, NewRequest(t, "GET", "/user1?tab=following"), http.StatusOK)
	page = NewHTMLParser(t, response.Body)

	// Verify it now follows follows 1 user
	testSelectorEquals(t, page, followingLink, "1 following")
	testSelectorEquals(t, page, listHeader, "Following")
	testListCount(t, page, listItems, followCount)
	page.AssertElement(t, listMsg, false)

	// Make user1 and user3 follow each other
	testUserFollowUser(t, user1, "user5")
	testUserFollowUser(t, user5, "user1")

	// = 2 follows =

	followCount++

	// Request the profile of user1, the Followers tab
	response = user1.MakeRequest(t, NewRequest(t, "GET", "/user1?tab=followers"), http.StatusOK)
	page = NewHTMLParser(t, response.Body)

	// Verify it is now followed by 2 users
	testSelectorEquals(t, page, followersLink, "2 followers")
	testSelectorEquals(t, page, listHeader, "Followers")
	testListCount(t, page, listItems, followCount)
	page.AssertElement(t, listMsg, false)

	// Request the profile of user1, the Following tab
	response = user1.MakeRequest(t, NewRequest(t, "GET", "/user1?tab=following"), http.StatusOK)
	page = NewHTMLParser(t, response.Body)

	// Verify it now follows follows 2 users
	testSelectorEquals(t, page, followingLink, "2 following")
	testSelectorEquals(t, page, listHeader, "Following")
	testListCount(t, page, listItems, followCount)
	page.AssertElement(t, listMsg, false)
}

// testUserFollowUser simply follows a user `following` by session of user `follower`
func testUserFollowUser(t *testing.T, follower *TestSession, following string) {
	t.Helper()
	follower.MakeRequest(t, NewRequestWithValues(t, "POST", fmt.Sprintf("/%s?action=follow", following),
		map[string]string{
			"_csrf": GetCSRF(t, follower, fmt.Sprintf("/%s", following)),
		}), http.StatusOK)
}

// testSelectorEquals prevents duplication of a lot of code for tests with many checks
func testSelectorEquals(t *testing.T, page *HTMLDoc, selector, expectedContent string) {
	t.Helper()
	element := page.Find(selector)
	content := strings.TrimSpace(element.Text())
	assert.Equal(t, expectedContent, content)
}

// testListCount checks that the list on the page has the right amount of items
func testListCount(t *testing.T, page *HTMLDoc, selector string, expectedCount int) {
	t.Helper()
	itemCount := page.Find(selector).Children().Length()
	assert.Equal(t, expectedCount, itemCount)
}
