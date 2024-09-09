// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
)

func TestRepoMigrationUI(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, giteaURL *url.URL) {
		sessionUser1 := loginUser(t, "user1")
		// Nothing is tested in plain Git migration form right now
		testRepoMigrationFormGitHub(t, sessionUser1)
		testRepoMigrationFormGitea(t, sessionUser1)
		testRepoMigrationFormGitLab(t, sessionUser1)
		testRepoMigrationFormGogs(t, sessionUser1)
		testRepoMigrationFormOneDev(t, sessionUser1)
		testRepoMigrationFormGitBucket(t, sessionUser1)
		testRepoMigrationFormCodebase(t, sessionUser1)
		testRepoMigrationFormForgejo(t, sessionUser1)
	})
}

func testRepoMigrationFormGitHub(t *testing.T, session *TestSession) {
	response := session.MakeRequest(t, NewRequest(t, "GET", "/repo/migrate?service_type=2"), http.StatusOK)
	page := NewHTMLParser(t, response.Body)

	items := page.Find("#migrate_items .field .checkbox input")
	expectedItems := []string{"issues", "pull_requests", "labels", "milestones", "releases"}
	testRepoMigrationFormItems(t, items, expectedItems)
}

func testRepoMigrationFormGitea(t *testing.T, session *TestSession) {
	response := session.MakeRequest(t, NewRequest(t, "GET", "/repo/migrate?service_type=3"), http.StatusOK)
	page := NewHTMLParser(t, response.Body)

	items := page.Find("#migrate_items .field .checkbox input")
	expectedItems := []string{"issues", "pull_requests", "labels", "milestones", "releases"}
	testRepoMigrationFormItems(t, items, expectedItems)
}

func testRepoMigrationFormGitLab(t *testing.T, session *TestSession) {
	response := session.MakeRequest(t, NewRequest(t, "GET", "/repo/migrate?service_type=4"), http.StatusOK)
	page := NewHTMLParser(t, response.Body)

	items := page.Find("#migrate_items .field .checkbox input")
	// Note: the checkbox "Merge requests" has name "pull_requests"
	expectedItems := []string{"issues", "pull_requests", "labels", "milestones", "releases"}
	testRepoMigrationFormItems(t, items, expectedItems)
}

func testRepoMigrationFormGogs(t *testing.T, session *TestSession) {
	response := session.MakeRequest(t, NewRequest(t, "GET", "/repo/migrate?service_type=5"), http.StatusOK)
	page := NewHTMLParser(t, response.Body)

	items := page.Find("#migrate_items .field .checkbox input")
	expectedItems := []string{"issues", "labels", "milestones"}
	testRepoMigrationFormItems(t, items, expectedItems)
}

func testRepoMigrationFormOneDev(t *testing.T, session *TestSession) {
	response := session.MakeRequest(t, NewRequest(t, "GET", "/repo/migrate?service_type=6"), http.StatusOK)
	page := NewHTMLParser(t, response.Body)

	items := page.Find("#migrate_items .field .checkbox input")
	expectedItems := []string{"issues", "pull_requests", "labels", "milestones"}
	testRepoMigrationFormItems(t, items, expectedItems)
}

func testRepoMigrationFormGitBucket(t *testing.T, session *TestSession) {
	response := session.MakeRequest(t, NewRequest(t, "GET", "/repo/migrate?service_type=7"), http.StatusOK)
	page := NewHTMLParser(t, response.Body)

	items := page.Find("#migrate_items .field .checkbox input")
	expectedItems := []string{"issues", "pull_requests", "labels", "milestones", "releases"}
	testRepoMigrationFormItems(t, items, expectedItems)
}

func testRepoMigrationFormCodebase(t *testing.T, session *TestSession) {
	response := session.MakeRequest(t, NewRequest(t, "GET", "/repo/migrate?service_type=8"), http.StatusOK)
	page := NewHTMLParser(t, response.Body)

	items := page.Find("#migrate_items .field .checkbox input")
	// Note: the checkbox "Merge requests" has name "pull_requests"
	expectedItems := []string{"issues", "pull_requests", "labels", "milestones"}
	testRepoMigrationFormItems(t, items, expectedItems)
}

func testRepoMigrationFormForgejo(t *testing.T, session *TestSession) {
	response := session.MakeRequest(t, NewRequest(t, "GET", "/repo/migrate?service_type=9"), http.StatusOK)
	page := NewHTMLParser(t, response.Body)

	items := page.Find("#migrate_items .field .checkbox input")
	expectedItems := []string{"issues", "pull_requests", "labels", "milestones", "releases"}
	testRepoMigrationFormItems(t, items, expectedItems)
}

func testRepoMigrationFormItems(t *testing.T, items *goquery.Selection, expectedItems []string) {
	t.Helper()

	// Compare lengths of item lists
	assert.EqualValues(t, len(expectedItems), items.Length())

	// Compare contents of item lists
	for index, expectedName := range expectedItems {
		name, exists := items.Eq(index).Attr("name")
		assert.True(t, exists)
		assert.EqualValues(t, expectedName, name)
	}
}
