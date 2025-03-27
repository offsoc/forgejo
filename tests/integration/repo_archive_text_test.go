// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"net/url"
	"path"
	"strings"
	"testing"

	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/translation"
	"forgejo.org/tests"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
)

func TestArchiveText(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, giteaURL *url.URL) {
		testUser := "user2"
		user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: testUser})
		session := loginUser(t, testUser)
		testRepoName := "archived_repo"
		tr := translation.NewLocale("en-US")
		link := path.Join(testUser, testRepoName, "settings")

		// Create test repo
		_, _, f := tests.CreateDeclarativeRepo(t, user2, testRepoName, nil, nil, nil)
		defer f()

		// Test settings page
		req := NewRequest(t, "GET", link)
		resp := session.MakeRequest(t, req, http.StatusOK)
		archivation := NewHTMLParser(t, resp.Body)
		testRepoArchiveElements(t, tr, archivation, "archive")

		// Archive repo
		req = NewRequestWithValues(t, "POST", link, map[string]string{
			"action": "archive",
			"_csrf":  GetCSRF(t, session, link),
		})
		_ = session.MakeRequest(t, req, http.StatusSeeOther)

		// Test settings page again
		req = NewRequest(t, "GET", link)
		resp = session.MakeRequest(t, req, http.StatusOK)
		unarchivation := NewHTMLParser(t, resp.Body)
		testRepoArchiveElements(t, tr, unarchivation, "unarchive")
	})
}

func testRepoArchiveElements(t *testing.T, tr translation.Locale, doc *HTMLDoc, opType string) {
	t.Helper()

	// Test danger section
	section := doc.Find(".danger.segment .flex-list .flex-item:has(.button[data-modal='#archive-repo-modal'])")
	testRepoArchiveElement(t, tr, section, ".flex-item-title", opType+".header")
	testRepoArchiveElement(t, tr, section, ".flex-item-body", opType+".text")
	testRepoArchiveElement(t, tr, section, ".button", opType+".button")

	// Test modal
	modal := doc.Find("#archive-repo-modal")
	testRepoArchiveElement(t, tr, modal, ".header", opType+".header")
	testRepoArchiveElement(t, tr, modal, ".message", opType+".text")
	testRepoArchiveElement(t, tr, modal, ".button.red", opType+".button")
}

func testRepoArchiveElement(t *testing.T, tr translation.Locale, doc *goquery.Selection, selector, op string) {
	t.Helper()

	element := doc.Find(selector).Text()
	element = strings.TrimSpace(element)
	assert.Equal(t, tr.TrString("repo.settings."+op), element)
}
