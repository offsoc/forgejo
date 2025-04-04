// Copyright 2024-2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"forgejo.org/models/db"
	unit_model "forgejo.org/models/unit"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	issue_service "forgejo.org/services/issue"
	files_service "forgejo.org/services/repository/files"
	"forgejo.org/tests"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserDashboardFeedWelcome(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// User2 has some activity in feed
	session := loginUser(t, "user2")
	page := NewHTMLParser(t, session.MakeRequest(t, NewRequest(t, "GET", "/"), http.StatusOK).Body)
	testUserDashboardFeedType(t, page, false)

	// User1 doesn't have any activity in feed
	session = loginUser(t, "user1")
	page = NewHTMLParser(t, session.MakeRequest(t, NewRequest(t, "GET", "/"), http.StatusOK).Body)
	testUserDashboardFeedType(t, page, true)
}

func testUserDashboardFeedType(t *testing.T, page *HTMLDoc, isEmpty bool) {
	page.AssertElement(t, "#activity-feed", !isEmpty)
	page.AssertElement(t, "#empty-feed", isEmpty)
}

func TestDashboardTitleRendering(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		user4 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 4})
		sess := loginUser(t, user4.Name)

		repo, _, f := tests.CreateDeclarativeRepo(t, user4, "",
			[]unit_model.Type{unit_model.TypePullRequests, unit_model.TypeIssues}, nil,
			[]*files_service.ChangeRepoFile{
				{
					Operation:     "create",
					TreePath:      "test.txt",
					ContentReader: strings.NewReader("Just some text here"),
				},
			},
		)
		defer f()

		issue := createIssue(t, user4, repo, "`:exclamation:` not rendered", "Hi there!")
		pr := createPullRequest(t, user4, repo, "testing", "`:exclamation:` not rendered")

		_, err := issue_service.CreateIssueComment(db.DefaultContext, user4, repo, issue, "hi", nil)
		require.NoError(t, err)

		_, err = issue_service.CreateIssueComment(db.DefaultContext, user4, repo, pr.Issue, "hi", nil)
		require.NoError(t, err)

		testIssueClose(t, sess, repo.OwnerName, repo.Name, strconv.Itoa(int(issue.Index)), false)
		testIssueClose(t, sess, repo.OwnerName, repo.Name, strconv.Itoa(int(pr.Issue.Index)), true)

		response := sess.MakeRequest(t, NewRequest(t, "GET", "/"), http.StatusOK)
		htmlDoc := NewHTMLParser(t, response.Body)

		count := 0
		htmlDoc.doc.Find("#activity-feed .flex-item-main .title").Each(func(i int, s *goquery.Selection) {
			count++
			if s.IsMatcher(goquery.Single("a")) {
				assert.Equal(t, "❗ not rendered", s.Text())
			} else {
				assert.Equal(t, ":exclamation: not rendered", s.Text())
			}
		})

		assert.Equal(t, 6, count)
	})
}
