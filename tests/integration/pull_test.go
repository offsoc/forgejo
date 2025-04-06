// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"testing"

	"forgejo.org/models/db"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unit"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestViewPulls(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	req := NewRequest(t, "GET", "/user2/repo1/pulls")
	resp := MakeRequest(t, req, http.StatusOK)

	htmlDoc := NewHTMLParser(t, resp.Body)
	search := htmlDoc.doc.Find(".list-header-search > .search > .input > input")
	placeholder, _ := search.Attr("placeholder")
	assert.Equal(t, "Search pulls…", placeholder)
}

func TestPullManuallyMergeWarning(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	session := loginUser(t, user2.Name)

	warningMessage := `Warning: The "Autodetect manual merge" setting is not enabled for this repository, you will have to mark this pull request as manually merged afterwards.`
	t.Run("Autodetect disabled", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, "GET", "/user2/repo1/pulls/3")
		resp := session.MakeRequest(t, req, http.StatusOK)

		htmlDoc := NewHTMLParser(t, resp.Body)
		mergeInstructions := htmlDoc.Find("#merge-instructions").Text()
		assert.Contains(t, mergeInstructions, warningMessage)
	})

	pullRequestUnit := unittest.AssertExistsAndLoadBean(t, &repo_model.RepoUnit{RepoID: 1, Type: unit.TypePullRequests})
	config := pullRequestUnit.PullRequestsConfig()
	config.AutodetectManualMerge = true
	_, err := db.GetEngine(db.DefaultContext).ID(pullRequestUnit.ID).Cols("config").Update(pullRequestUnit)
	require.NoError(t, err)

	t.Run("Autodetect enabled", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, "GET", "/user2/repo1/pulls/3")
		resp := session.MakeRequest(t, req, http.StatusOK)

		htmlDoc := NewHTMLParser(t, resp.Body)
		mergeInstructions := htmlDoc.Find("#merge-instructions").Text()
		assert.NotContains(t, mergeInstructions, warningMessage)
	})
}

func TestPullCombinedReviewRequest(t *testing.T) {
	defer tests.AddFixtures("tests/integration/fixtures/TestPullCombinedReviewRequest/")()
	defer tests.PrepareTestEnv(t)()

	session := loginUser(t, "user2")

	helper := func(t *testing.T, action, userID, expectedText string) {
		t.Helper()

		req := NewRequestWithValues(t, "POST", "/user2/repo1/pulls/request_review", map[string]string{
			"_csrf":     GetCSRF(t, session, "/user2/repo1/pulls/3"),
			"issue_ids": "3",
			"action":    action,
			"id":        userID,
		})
		session.MakeRequest(t, req, http.StatusOK)

		req = NewRequest(t, "GET", "/user2/repo1/pulls/3")
		resp := session.MakeRequest(t, req, http.StatusOK)
		htmlDoc := NewHTMLParser(t, resp.Body)

		assert.Contains(t, htmlDoc.Find(".timeline-item:has(.review-request-list)").Last().Text(), expectedText)
	}

	helper(t, "detach", "2", "refused to review")
	helper(t, "attach", "4", "requested reviews from user4 and removed review requests for user2")
	helper(t, "attach", "9", "requested reviews from user4, user9 and removed review requests for user2")
	helper(t, "attach", "2", "requested reviews from user4, user9")
	helper(t, "detach", "4", "requested review from user9")
	helper(t, "detach", "11", "requested reviews from user9 and removed review requests for user11")
	helper(t, "detach", "9", "removed review request for user11")
	helper(t, "detach", "2", "removed review requests for user11, user2")
}
