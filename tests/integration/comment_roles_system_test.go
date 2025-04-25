// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"net/http"
	"testing"

	issues_model "forgejo.org/models/issues"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
)

// TestSystemCommentRoles verifies that system users don't have role labels.
// As it is not possible to do actions as system users, the tests are done using fixtures.

func TestSystemCommentRoles(t *testing.T) {
	defer unittest.OverrideFixtures("tests/integration/fixtures/TestSystemCommentRoles")()
	defer tests.PrepareTestEnv(t)()

	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})

	testCases := []struct {
		name      string
		username  string
		index     int64
		roleCount int64
	}{
		{"user2", "user2", 1000, 1}, // As a verification, also check a normal user with one role.
		{"Ghost", "Ghost", 1001, 0}, // System users should not have any roles, so 0.
		{"Actions", "forgejo-actions", 1002, 0},
		{"APActor", "Ghost", 1003, 0}, // actor is displayed as Ghost, could be a bug.
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{
				RepoID: repo.ID,
				Index:  tc.index,
			})

			req := NewRequestf(t, "GET", "%s/issues/%d", repo.Link(), issue.Index)
			resp := MakeRequest(t, req, http.StatusOK)
			htmlDoc := NewHTMLParser(t, resp.Body)

			assert.Equal(t, tc.username, htmlDoc.Find("a.author").Text())
			assert.EqualValues(t, tc.roleCount, htmlDoc.Find(".role-label").Length())
		})
	}
}
