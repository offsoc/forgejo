// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"net/http"
	"strings"
	"testing"

	org_model "forgejo.org/models/organization"
	project_model "forgejo.org/models/project"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
)

func TestPrivateIssueProject(t *testing.T) {
	defer unittest.OverrideFixtures("models/fixtures/PrivateIssueProjects")()
	defer tests.PrepareTestEnv(t)()

	user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	sess := loginUser(t, user2.Name)

	test := func(t *testing.T, sess *TestSession, username string, projectID int64, hasAccess bool, publicIssueHref ...string) {
		t.Helper()
		defer tests.PrintCurrentTest(t, 1)()

		// Test that the projects overview page shows the correct open and close issues.
		req := NewRequestf(t, "GET", "%s/-/projects", username)
		resp := sess.MakeRequest(t, req, http.StatusOK)

		htmlDoc := NewHTMLParser(t, resp.Body)
		openCloseStats := htmlDoc.Find(".milestone-toolbar .group").First().Text()
		if hasAccess {
			assert.Contains(t, openCloseStats, "2\u00a0Open")
		} else {
			assert.Contains(t, openCloseStats, "1\u00a0Open")
		}
		assert.Contains(t, openCloseStats, "0\u00a0Closed")

		// Check that on the project itself the issue is not shown.
		req = NewRequestf(t, "GET", "%s/-/projects/%d", username, projectID)
		resp = sess.MakeRequest(t, req, http.StatusOK)

		htmlDoc = NewHTMLParser(t, resp.Body)
		issueCardsLen := htmlDoc.Find(".project-column .issue-card").Length()
		if hasAccess {
			assert.Equal(t, 2, issueCardsLen)
		} else {
			assert.Equal(t, 1, issueCardsLen)
			// Ensure that the public issue is shown.
			assert.Equal(t, publicIssueHref[0], htmlDoc.Find(".project-column .issue-card .issue-card-title").AttrOr("href", ""))
		}

		// And that the issue count is correct.
		issueCount := strings.TrimSpace(htmlDoc.Find(".project-column-issue-count").Text())
		if hasAccess {
			assert.Equal(t, "2", issueCount)
		} else {
			assert.Equal(t, "1", issueCount)
		}
	}

	t.Run("Organization project", func(t *testing.T) {
		org := unittest.AssertExistsAndLoadBean(t, &org_model.Organization{ID: 3})
		orgProject := unittest.AssertExistsAndLoadBean(t, &project_model.Project{ID: 1001, OwnerID: org.ID})

		t.Run("Authenticated user", func(t *testing.T) {
			test(t, sess, org.Name, orgProject.ID, true)
		})

		t.Run("Anonymous user", func(t *testing.T) {
			test(t, emptyTestSession(t), org.Name, orgProject.ID, false, "/org3/repo21/issues/1")
		})
	})

	t.Run("User project", func(t *testing.T) {
		userProject := unittest.AssertExistsAndLoadBean(t, &project_model.Project{ID: 1002, OwnerID: user2.ID})

		t.Run("Authenticated user", func(t *testing.T) {
			test(t, sess, user2.Name, userProject.ID, true)
		})

		t.Run("Anonymous user", func(t *testing.T) {
			test(t, emptyTestSession(t), user2.Name, userProject.ID, false, "/user2/repo1/issues/1")
		})
	})
}
