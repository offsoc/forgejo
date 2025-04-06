// Copyright 2024 The Forgejo Authors c/o Codeberg e.V.. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"testing"

	auth_model "forgejo.org/models/auth"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	api "forgejo.org/modules/structs"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
)

func TestAPIRepoActivitiyFeeds(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
	repo, _, f := tests.CreateDeclarativeRepoWithOptions(t, owner, tests.DeclarativeRepoOptions{})
	defer f()

	feedURL := fmt.Sprintf("/api/v1/repos/%s/activities/feeds", repo.FullName())
	assertAndReturnActivities := func(t *testing.T, length int) []api.Activity {
		t.Helper()

		req := NewRequest(t, "GET", feedURL)
		resp := MakeRequest(t, req, http.StatusOK)
		var activities []api.Activity
		DecodeJSON(t, resp, &activities)

		assert.Len(t, activities, length)

		return activities
	}
	createIssue := func(t *testing.T) {
		session := loginUser(t, owner.Name)
		token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteIssue)
		urlStr := fmt.Sprintf("/api/v1/repos/%s/issues?state=all", repo.FullName())
		req := NewRequestWithJSON(t, "POST", urlStr, &api.CreateIssueOption{
			Title: "ActivityFeed test",
			Body:  "Nothing to see here!",
		}).AddTokenAuth(token)
		MakeRequest(t, req, http.StatusCreated)
	}

	t.Run("repo creation", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		// Upon repo creation, there's a single activity.
		assertAndReturnActivities(t, 1)
	})

	t.Run("single watcher, single issue", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		// After creating an issue, we'll have two activities.
		createIssue(t)
		assertAndReturnActivities(t, 2)
	})

	t.Run("a new watcher, no new activities", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		watcher := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
		watcherSession := loginUser(t, watcher.Name)
		watcherToken := getTokenForLoggedInUser(t, watcherSession, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeReadUser)

		req := NewRequest(t, "PUT", fmt.Sprintf("/api/v1/repos/%s/subscription", repo.FullName())).
			AddTokenAuth(watcherToken)
		MakeRequest(t, req, http.StatusOK)

		assertAndReturnActivities(t, 2)
	})

	t.Run("multiple watchers, new issue", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		// After creating a second issue, we'll have three activities, even
		// though we have multiple watchers.
		createIssue(t)
		assertAndReturnActivities(t, 3)
	})
}
