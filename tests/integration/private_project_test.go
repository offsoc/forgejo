// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"net/http"
	"strings"
	"testing"

	org_model "code.gitea.io/gitea/models/organization"
	project_model "code.gitea.io/gitea/models/project"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/tests"

	"github.com/stretchr/testify/assert"
)

func TestPrivateIssueProject(t *testing.T) {
	defer tests.AddFixtures("models/fixtures/PrivateIssueProjects/")()
	defer tests.PrepareTestEnv(t)()

	user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	sess := loginUser(t, user2.Name)

	test := func(t *testing.T, sess *TestSession, username string, projectID int64, hasAccess bool) {
		t.Helper()
		defer tests.PrintCurrentTest(t, 1)()

		// Test that the projects overview page shows the correct open and close issues.
		req := NewRequestf(t, "GET", "%s/-/projects", username)
		resp := sess.MakeRequest(t, req, http.StatusOK)

		htmlDoc := NewHTMLParser(t, resp.Body)
		openCloseStats := htmlDoc.Find(".milestone-toolbar .group").First().Text()
		if hasAccess {
			assert.Contains(t, openCloseStats, "1\u00a0Open")
		} else {
			assert.Contains(t, openCloseStats, "0\u00a0Open")
		}
		assert.Contains(t, openCloseStats, "0\u00a0Closed")

		// Check that on the project itself the issue is not shown.
		req = NewRequestf(t, "GET", "%s/-/projects/%d", username, projectID)
		resp = sess.MakeRequest(t, req, http.StatusOK)

		htmlDoc = NewHTMLParser(t, resp.Body)
		htmlDoc.AssertElement(t, ".project-column .issue-card", hasAccess)

		// And that the issue count is correct.
		issueCount := strings.TrimSpace(htmlDoc.Find(".project-column-issue-count").Text())
		if hasAccess {
			assert.EqualValues(t, "1", issueCount)
		} else {
			assert.EqualValues(t, "0", issueCount)
		}
	}

	t.Run("Organization project", func(t *testing.T) {
		org := unittest.AssertExistsAndLoadBean(t, &org_model.Organization{ID: 3})
		orgProject := unittest.AssertExistsAndLoadBean(t, &project_model.Project{ID: 1001, OwnerID: org.ID})

		t.Run("Authenticated user", func(t *testing.T) {
			test(t, sess, org.Name, orgProject.ID, true)
		})

		t.Run("Anonymous user", func(t *testing.T) {
			test(t, emptyTestSession(t), org.Name, orgProject.ID, false)
		})
	})

	t.Run("User project", func(t *testing.T) {
		userProject := unittest.AssertExistsAndLoadBean(t, &project_model.Project{ID: 1002, OwnerID: user2.ID})

		t.Run("Authenticated user", func(t *testing.T) {
			test(t, sess, user2.Name, userProject.ID, true)
		})

		t.Run("Anonymous user", func(t *testing.T) {
			test(t, emptyTestSession(t), user2.Name, userProject.ID, false)
		})
	})
}
