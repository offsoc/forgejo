// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"testing"

	"forgejo.org/tests"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
)

// TestRepoMigrationUI is used to test various form properties of different migration types
func TestRepoMigrationUI(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	session := loginUser(t, "user1")
	// Note: nothing is tested in plain Git migration form right now

	t.Run("GitHub", func(t *testing.T) {
		response := session.MakeRequest(t, NewRequest(t, "GET", "/repo/migrate?service_type=2"), http.StatusOK)
		page := NewHTMLParser(t, response.Body)

		items := page.Find("#migrate_items .field .checkbox input")
		expectedItems := []string{"issues", "pull_requests", "labels", "milestones", "releases"}
		testRepoMigrationFormItems(t, items, expectedItems)
	})
	t.Run("Gitea", func(t *testing.T) {
		response := session.MakeRequest(t, NewRequest(t, "GET", "/repo/migrate?service_type=3"), http.StatusOK)
		page := NewHTMLParser(t, response.Body)

		items := page.Find("#migrate_items .field .checkbox input")
		expectedItems := []string{"issues", "pull_requests", "labels", "milestones", "releases"}
		testRepoMigrationFormItems(t, items, expectedItems)
	})
	t.Run("GitLab", func(t *testing.T) {
		response := session.MakeRequest(t, NewRequest(t, "GET", "/repo/migrate?service_type=4"), http.StatusOK)
		page := NewHTMLParser(t, response.Body)

		items := page.Find("#migrate_items .field .checkbox input")
		// Note: the checkbox "Merge requests" has name "pull_requests"
		expectedItems := []string{"issues", "pull_requests", "labels", "milestones", "releases"}
		testRepoMigrationFormItems(t, items, expectedItems)
	})
	t.Run("Gogs", func(t *testing.T) {
		response := session.MakeRequest(t, NewRequest(t, "GET", "/repo/migrate?service_type=5"), http.StatusOK)
		page := NewHTMLParser(t, response.Body)

		items := page.Find("#migrate_items .field .checkbox input")
		expectedItems := []string{"issues", "labels", "milestones"}
		testRepoMigrationFormItems(t, items, expectedItems)
	})
	t.Run("OneDev", func(t *testing.T) {
		response := session.MakeRequest(t, NewRequest(t, "GET", "/repo/migrate?service_type=6"), http.StatusOK)
		page := NewHTMLParser(t, response.Body)

		items := page.Find("#migrate_items .field .checkbox input")
		expectedItems := []string{"issues", "pull_requests", "labels", "milestones"}
		testRepoMigrationFormItems(t, items, expectedItems)
	})
	t.Run("GitBucket", func(t *testing.T) {
		response := session.MakeRequest(t, NewRequest(t, "GET", "/repo/migrate?service_type=7"), http.StatusOK)
		page := NewHTMLParser(t, response.Body)

		items := page.Find("#migrate_items .field .checkbox input")
		expectedItems := []string{"issues", "pull_requests", "labels", "milestones", "releases"}
		testRepoMigrationFormItems(t, items, expectedItems)
	})
	t.Run("Codebase", func(t *testing.T) {
		response := session.MakeRequest(t, NewRequest(t, "GET", "/repo/migrate?service_type=8"), http.StatusOK)
		page := NewHTMLParser(t, response.Body)

		items := page.Find("#migrate_items .field .checkbox input")
		// Note: the checkbox "Merge requests" has name "pull_requests"
		expectedItems := []string{"issues", "pull_requests", "labels", "milestones"}
		testRepoMigrationFormItems(t, items, expectedItems)
	})
	t.Run("Codebase", func(t *testing.T) {
		response := session.MakeRequest(t, NewRequest(t, "GET", "/repo/migrate?service_type=9"), http.StatusOK)
		page := NewHTMLParser(t, response.Body)

		items := page.Find("#migrate_items .field .checkbox input")
		expectedItems := []string{"issues", "pull_requests", "labels", "milestones", "releases"}
		testRepoMigrationFormItems(t, items, expectedItems)
	})
}

func testRepoMigrationFormItems(t *testing.T, items *goquery.Selection, expectedItems []string) {
	t.Helper()

	// Compare lengths of item lists
	assert.Equal(t, len(expectedItems), items.Length())

	// Compare contents of item lists
	for index, expectedName := range expectedItems {
		name, exists := items.Eq(index).Attr("name")
		assert.True(t, exists)
		assert.Equal(t, expectedName, name)
	}
}
