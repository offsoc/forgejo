// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	unit_model "forgejo.org/models/unit"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/test"
	files_service "forgejo.org/services/repository/files"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
)

func TestRenderFileSVGIsInImgTag(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	session := loginUser(t, "user2")

	req := NewRequest(t, "GET", "/user2/repo2/src/branch/master/line.svg")
	resp := session.MakeRequest(t, req, http.StatusOK)

	doc := NewHTMLParser(t, resp.Body)
	src, exists := doc.doc.Find(".file-view img").Attr("src")
	assert.True(t, exists, "The SVG image should be in an <img> tag so that scripts in the SVG are not run")
	assert.Equal(t, "/user2/repo2/raw/branch/master/line.svg", src)
}

func TestAmbiguousCharacterDetection(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
		session := loginUser(t, user2.Name)

		// Prepare the environments. File view, commit view (diff), wiki page.
		repo, commitID, f := tests.CreateDeclarativeRepo(t, user2, "",
			[]unit_model.Type{unit_model.TypeCode, unit_model.TypeWiki}, nil,
			[]*files_service.ChangeRepoFile{
				{
					Operation:     "create",
					TreePath:      "test.sh",
					ContentReader: strings.NewReader("Hello there!\nline western"),
				},
			},
		)
		defer f()

		req := NewRequestWithValues(t, "POST", repo.Link()+"/wiki?action=new", map[string]string{
			"_csrf":   GetCSRF(t, session, repo.Link()+"/wiki?action=new"),
			"title":   "Normal",
			"content": "Hello – Hello",
		})
		session.MakeRequest(t, req, http.StatusSeeOther)

		assertCase := func(t *testing.T, fileContext, commitContext, wikiContext bool) {
			t.Helper()

			t.Run("File context", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				req := NewRequest(t, "GET", repo.Link()+"/src/branch/main/test.sh")
				resp := session.MakeRequest(t, req, http.StatusOK)

				htmlDoc := NewHTMLParser(t, resp.Body)
				htmlDoc.AssertElement(t, ".unicode-escape-prompt", fileContext)
			})
			t.Run("Commit context", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				req := NewRequest(t, "GET", repo.Link()+"/commit/"+commitID)
				resp := session.MakeRequest(t, req, http.StatusOK)

				htmlDoc := NewHTMLParser(t, resp.Body)
				htmlDoc.AssertElement(t, ".lines-escape .toggle-escape-button", commitContext)
			})
			t.Run("Wiki context", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				req := NewRequest(t, "GET", repo.Link()+"/wiki/Normal")
				resp := session.MakeRequest(t, req, http.StatusOK)

				htmlDoc := NewHTMLParser(t, resp.Body)
				htmlDoc.AssertElement(t, ".unicode-escape-prompt", wikiContext)
			})
		}

		t.Run("Enabled all context", func(t *testing.T) {
			defer test.MockVariableValue(&setting.UI.SkipEscapeContexts, []string{})()

			assertCase(t, true, true, true)
		})

		t.Run("Enabled file context", func(t *testing.T) {
			defer test.MockVariableValue(&setting.UI.SkipEscapeContexts, []string{"diff", "wiki"})()

			assertCase(t, true, false, false)
		})

		t.Run("Enabled commit context", func(t *testing.T) {
			defer test.MockVariableValue(&setting.UI.SkipEscapeContexts, []string{"file-view", "wiki"})()

			assertCase(t, false, true, false)
		})

		t.Run("Enabled wiki context", func(t *testing.T) {
			defer test.MockVariableValue(&setting.UI.SkipEscapeContexts, []string{"file-view", "diff"})()

			assertCase(t, false, false, true)
		})

		t.Run("No context", func(t *testing.T) {
			defer test.MockVariableValue(&setting.UI.SkipEscapeContexts, []string{"file-view", "wiki", "diff"})()

			assertCase(t, false, false, false)
		})

		t.Run("Disabled detection", func(t *testing.T) {
			defer test.MockVariableValue(&setting.UI.SkipEscapeContexts, []string{})()
			defer test.MockVariableValue(&setting.UI.AmbiguousUnicodeDetection, false)()

			assertCase(t, false, false, false)
		})
	})
}

func TestCommitListActions(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
		session := loginUser(t, user2.Name)
		repo, commitID, f := tests.CreateDeclarativeRepo(t, user2, "",
			[]unit_model.Type{unit_model.TypeCode, unit_model.TypeWiki}, nil,
			[]*files_service.ChangeRepoFile{
				{
					Operation:     "create",
					TreePath:      "test.sh",
					ContentReader: strings.NewReader("Hello there!"),
				},
			},
		)
		defer f()

		req := NewRequestWithValues(t, "POST", repo.Link()+"/wiki?action=new", map[string]string{
			"_csrf":   GetCSRF(t, session, repo.Link()+"/wiki?action=new"),
			"title":   "Normal",
			"content": "Hello world!",
		})
		session.MakeRequest(t, req, http.StatusSeeOther)

		t.Run("Wiki revision", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequest(t, "GET", repo.Link()+"/wiki/Normal?action=_revision")
			resp := session.MakeRequest(t, req, http.StatusOK)
			htmlDoc := NewHTMLParser(t, resp.Body)

			htmlDoc.AssertElement(t, fmt.Sprintf(".commit-list a[href^='/%s/src/commit/']", repo.FullName()), false)
		})

		fileDiffSelector := fmt.Sprintf(".commit-list a[href='/%s/commit/%s?files=test.sh']", repo.FullName(), commitID)
		t.Run("Commit list", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequest(t, "GET", repo.Link()+"/commits/branch/main")
			resp := session.MakeRequest(t, req, http.StatusOK)
			htmlDoc := NewHTMLParser(t, resp.Body)

			htmlDoc.AssertElement(t, fmt.Sprintf(".commit-list a[href='/%s/src/commit/%s']", repo.FullName(), commitID), true)
			htmlDoc.AssertElement(t, fileDiffSelector, false)
		})

		t.Run("File history", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequest(t, "GET", repo.Link()+"/commits/branch/main/test.sh")
			resp := session.MakeRequest(t, req, http.StatusOK)
			htmlDoc := NewHTMLParser(t, resp.Body)

			htmlDoc.AssertElement(t, fmt.Sprintf(".commit-list a[href='/%s/src/commit/%s/test.sh']", repo.FullName(), commitID), true)
			htmlDoc.AssertElement(t, fileDiffSelector, true)
		})
	})
}

func TestTitleDisplayName(t *testing.T) {
	session := emptyTestSession(t)
	title := GetHTMLTitle(t, session, "/")
	assert.Equal(t, "Forgejo: Beyond coding. We Forge.", title)
}

func TestHomeDisplayName(t *testing.T) {
	session := emptyTestSession(t)
	req := NewRequest(t, "GET", "/")
	resp := session.MakeRequest(t, req, http.StatusOK)
	htmlDoc := NewHTMLParser(t, resp.Body)
	assert.Equal(t, "Forgejo: Beyond coding. We Forge.", strings.TrimSpace(htmlDoc.Find("h1.title").Text()))
}

func TestOpenGraphDisplayName(t *testing.T) {
	session := emptyTestSession(t)
	req := NewRequest(t, "GET", "/")
	resp := session.MakeRequest(t, req, http.StatusOK)
	htmlDoc := NewHTMLParser(t, resp.Body)
	ogTitle, _ := htmlDoc.Find("meta[property='og:title']").Attr("content")
	assert.Equal(t, "Forgejo: Beyond coding. We Forge.", ogTitle)
	ogSiteName, _ := htmlDoc.Find("meta[property='og:site_name']").Attr("content")
	assert.Equal(t, "Forgejo: Beyond coding. We Forge.", ogSiteName)
}
