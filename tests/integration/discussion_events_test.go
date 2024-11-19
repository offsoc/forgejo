// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"fmt"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
	"testing"

	auth_model "code.gitea.io/gitea/models/auth"
	api "code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/tests"

	"github.com/stretchr/testify/assert"
)

// TestDiscussionEvents is a test for various events displayed in the timelines of pulls and issues
func TestDiscussionEvents(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	repoName := "discussion-timeline-tests"
	user1 := "user1"
	description := "This PR will be used for testing events in discussions"
	// Expected branch name when initializing repo automatically
	defaultBranch := "master"
	htmlCleaner := regexp.MustCompile(`[\t\n]`)

	onGiteaRun(t, func(t *testing.T, giteaURL *url.URL) {
		sessionUser1 := loginUser(t, user1)
		tokenUser1 := getTokenForLoggedInUser(t, sessionUser1, auth_model.AccessTokenScopeAll)

		// Create test repo
		var repo api.Repository
		resp := sessionUser1.MakeRequest(t, NewRequestWithJSON(t, "POST", "/api/v1/user/repos", &api.CreateRepoOption{
			Name:     repoName,
			AutoInit: true,
		}).AddTokenAuth(tokenUser1), http.StatusCreated)
		DecodeJSON(t, resp, &repo)

		// == Test pulls ==

		// Open a new PR as user1
		testEditFileToNewBranch(t, sessionUser1, user1, repo.Name, defaultBranch, "comment-labels", "README.md", description)
		sessionUser1.MakeRequest(t, NewRequestWithValues(t, "POST", path.Join(repo.FullName, "compare", fmt.Sprintf("%s...comment-labels", defaultBranch)),
			map[string]string{
				"_csrf": GetCSRF(t, sessionUser1, path.Join(repo.FullName, "compare", fmt.Sprintf("%s...comment-labels", defaultBranch))),
				"title": description,
			},
		), http.StatusOK)

		// Pull number, expected to be 1 in a fresh repo
		testPullID := "1"

		// Get the PR page and find all events
		response := sessionUser1.MakeRequest(t, NewRequest(t, "GET", path.Join(repo.FullName, "pulls", testPullID)), http.StatusOK)
		page := NewHTMLParser(t, response.Body)
		events := page.Find(".timeline .timeline-item.event .text")

		// Check the event. Should contain: "<username> added 1 commit <relative-time>"
		event := events.Eq(0)
		eventHTML, _ := event.Html()
		eventText := htmlCleaner.ReplaceAllString(strings.TrimSpace(event.Text()), "")
		assert.Contains(t, eventHTML, `href="/user1">user1</a>`)
		assert.Contains(t, eventHTML, `<relative-time`)
		assert.Contains(t, eventText, `user1 added 1 commit`)
	})
}
