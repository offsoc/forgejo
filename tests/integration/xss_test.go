// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"testing"

	issues_model "forgejo.org/models/issues"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
)

func TestXSSUserFullName(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	const fullName = `name & <script class="evil">alert('Oh no!');</script>`

	session := loginUser(t, user.Name)
	req := NewRequestWithValues(t, "POST", "/user/settings", map[string]string{
		"_csrf":     GetCSRF(t, session, "/user/settings"),
		"name":      user.Name,
		"full_name": fullName,
		"email":     user.Email,
		"language":  "en-US",
	})
	session.MakeRequest(t, req, http.StatusSeeOther)

	req = NewRequestf(t, "GET", "/%s", user.Name)
	resp := session.MakeRequest(t, req, http.StatusOK)
	htmlDoc := NewHTMLParser(t, resp.Body)
	assert.Equal(t, 0, htmlDoc.doc.Find("script.evil").Length())
	assert.Equal(t, fullName,
		htmlDoc.doc.Find("div.content").Find(".header.text.center").Text(),
	)
}

func TestXSSWikiLastCommitInfo(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	// Check on page view.
	t.Run("Page view", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, http.MethodGet, "/user2/repo1/wiki/XSS")
		resp := MakeRequest(t, req, http.StatusOK)
		htmlDoc := NewHTMLParser(t, resp.Body)

		htmlDoc.AssertElement(t, "script.evil", false)
		assert.Contains(t, htmlDoc.Find(".ui.sub.header").Text(), `Gusted<script class="evil">alert('Oh no!');</script> edited this page 2024-01-31`)
	})

	// Check on revisions page.
	t.Run("Revision page", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, http.MethodGet, "/user2/repo1/wiki/XSS?action=_revision")
		resp := MakeRequest(t, req, http.StatusOK)
		htmlDoc := NewHTMLParser(t, resp.Body)

		htmlDoc.AssertElement(t, "script.evil", false)
		assert.Contains(t, htmlDoc.Find(".ui.sub.header").Text(), `Gusted<script class="evil">alert('Oh no!');</script> edited this page 2024-01-31`)
	})
}

func TestXSSReviewDismissed(t *testing.T) {
	defer unittest.OverrideFixtures("tests/integration/fixtures/TestXSSReviewDismissed")()
	defer tests.PrepareTestEnv(t)()

	review := unittest.AssertExistsAndLoadBean(t, &issues_model.Review{ID: 1000})

	req := NewRequest(t, http.MethodGet, fmt.Sprintf("/user2/repo1/pulls/%d", +review.IssueID))
	resp := MakeRequest(t, req, http.StatusOK)
	htmlDoc := NewHTMLParser(t, resp.Body)

	htmlDoc.AssertElement(t, "script.evil", false)
	assert.Contains(t, htmlDoc.Find("#issuecomment-1000 .dismissed-message").Text(), `dismissed Otto <script class='evil'>alert('Oh no!')</script>'s review`)
}
