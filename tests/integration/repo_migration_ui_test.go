// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"fmt"
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

	type Migration struct {
		Name                      string
		ExpectedItems             []string
		DescriptionHasPlaceholder bool
	}

	migrations := map[int]Migration{
		2: {
			"GitHub",
			[]string{"issues", "pull_requests", "labels", "milestones", "releases"},
			true,
		},
		3: {
			"Gitea",
			[]string{"issues", "pull_requests", "labels", "milestones", "releases"},
			true,
		},
		4: {
			"GitLab",
			// Note: the checkbox "Merge requests" has name "pull_requests"
			[]string{"issues", "pull_requests", "labels", "milestones", "releases"},
			true,
		},
		5: {
			"Gogs",
			[]string{"issues", "labels", "milestones"},
			true,
		},
		6: {
			"OneDev",
			[]string{"issues", "pull_requests", "labels", "milestones"},
			true,
		},
		7: {
			"GitBucket",
			[]string{"issues", "pull_requests", "labels", "milestones", "releases"},
			false,
		},
		8: {
			"Codebase",
			// Note: the checkbox "Merge requests" has name "pull_requests"
			[]string{"issues", "pull_requests", "labels", "milestones"},
			true,
		},
		9: {
			"Forgejo",
			[]string{"issues", "pull_requests", "labels", "milestones", "releases"},
			true,
		},
	}

	for id, migration := range migrations {
		t.Run(migration.Name, func(t *testing.T) {
			response := session.MakeRequest(t, NewRequest(t, "GET", fmt.Sprintf("/repo/migrate?service_type=%d", id)), http.StatusOK)
			page := NewHTMLParser(t, response.Body)

			items := page.Find("#migrate_items .field .checkbox input")
			testRepoMigrationFormItems(t, items, migration.ExpectedItems)

			page.AssertElement(t, "#clone_addr", true)
			autocomplete, _ := page.Find("#clone_addr").Attr("autocomplete")
			assert.Equal(t, "url", autocomplete)

			page.AssertElement(t, "#description", true)
			_, descriptionHasPlaceholder := page.Find("#description").Attr("placeholder")
			assert.Equal(t, migration.DescriptionHasPlaceholder, descriptionHasPlaceholder)
		})
	}
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
