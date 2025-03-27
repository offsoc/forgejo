// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	repo_model "forgejo.org/models/repo"
	unit_model "forgejo.org/models/unit"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	files_service "forgejo.org/services/repository/files"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
)

func TestCitation(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})

		session := loginUser(t, user.LoginName)

		t.Run("No citation", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			repo, _, f := tests.CreateDeclarativeRepo(t, user, "citation-no-citation", []unit_model.Type{unit_model.TypeCode}, nil, nil)
			defer f()

			testCitationButtonExists(t, session, repo, "", false)
		})

		t.Run("cff citation", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			repo, f := createRepoWithEmptyFile(t, user, "citation-cff", "CITATION.cff")
			defer f()

			testCitationButtonExists(t, session, repo, "CITATION.cff", true)
		})

		t.Run("bib citation", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			repo, f := createRepoWithEmptyFile(t, user, "citation-bib", "CITATION.bib")
			defer f()

			testCitationButtonExists(t, session, repo, "CITATION.bib", true)
		})
	})
}

func testCitationButtonExists(t *testing.T, session *TestSession, repo *repo_model.Repository, file string, exists bool) {
	req := NewRequest(t, "GET", repo.HTMLURL())
	resp := session.MakeRequest(t, req, http.StatusOK)
	doc := NewHTMLParser(t, resp.Body)

	doc.AssertElement(t, "#cite-repo-button", exists)

	if exists {
		href, exists := doc.doc.Find("#goto-citation-btn").Attr("href")
		assert.True(t, exists)

		assert.True(t, strings.HasSuffix(href, file))
	}
}

func createRepoWithEmptyFile(t *testing.T, user *user_model.User, repoName, fileName string) (*repo_model.Repository, func()) {
	repo, _, f := tests.CreateDeclarativeRepo(t, user, repoName, []unit_model.Type{unit_model.TypeCode}, nil, []*files_service.ChangeRepoFile{
		{
			Operation: "create",
			TreePath:  fileName,
		},
	})

	return repo, f
}
