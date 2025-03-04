// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"code.gitea.io/gitea/models/db"
	unit_model "code.gitea.io/gitea/models/unit"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/translation"
	issue_service "code.gitea.io/gitea/services/issue"
	files_service "code.gitea.io/gitea/services/repository/files"
	"code.gitea.io/gitea/tests"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserDashboardActionLinks(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	session := loginUser(t, "user1")
	locale := translation.NewLocale("en-US")

	response := session.MakeRequest(t, NewRequest(t, "GET", "/"), http.StatusOK)
	page := NewHTMLParser(t, response.Body)
	links := page.Find("#navbar .dropdown[data-tooltip-content='Create…'] .menu")
	assert.EqualValues(t, locale.TrString("new_repo.link"), strings.TrimSpace(links.Find("a[href='/repo/create']").Text()))
	assert.EqualValues(t, locale.TrString("new_migrate.link"), strings.TrimSpace(links.Find("a[href='/repo/migrate']").Text()))
	assert.EqualValues(t, locale.TrString("new_org.link"), strings.TrimSpace(links.Find("a[href='/org/create']").Text()))
}

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
			assert.EqualValues(t, ":exclamation: not rendered", s.Text())
		})

		assert.EqualValues(t, 6, count)
	})
}
