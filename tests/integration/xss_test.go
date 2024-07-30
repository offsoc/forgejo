// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	issues_model "code.gitea.io/gitea/models/issues"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/tests"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	assert.EqualValues(t, 0, htmlDoc.doc.Find("script.evil").Length())
	assert.EqualValues(t, fullName,
		htmlDoc.doc.Find("div.content").Find(".header.text.center").Text(),
	)
}

func TestXSSWikiLastCommitInfo(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		// Prepare the environment.
		dstPath := t.TempDir()
		r := fmt.Sprintf("%suser2/repo1.wiki.git", u.String())
		u, err := url.Parse(r)
		require.NoError(t, err)
		u.User = url.UserPassword("user2", userPassword)
		require.NoError(t, git.CloneWithArgs(context.Background(), git.AllowLFSFiltersArgs(), u.String(), dstPath, git.CloneRepoOptions{}))

		// Use go-git here, because using git wouldn't work, it has code to remove
		// `<`, `>` and `\n` in user names. Even though this is permitted and
		// wouldn't result in a error by a Git server.
		gitRepo, err := gogit.PlainOpen(dstPath)
		require.NoError(t, err)

		w, err := gitRepo.Worktree()
		require.NoError(t, err)

		filename := filepath.Join(dstPath, "Home.md")
		err = os.WriteFile(filename, []byte("Oh, a XSS attack?"), 0o644)
		require.NoError(t, err)

		_, err = w.Add("Home.md")
		require.NoError(t, err)

		_, err = w.Commit("Yay XSS", &gogit.CommitOptions{
			Author: &object.Signature{
				Name:  `Gusted<script class="evil">alert('Oh no!');</script>`,
				Email: "valid@example.org",
				When:  time.Date(2024, time.January, 31, 0, 0, 0, 0, time.UTC),
			},
		})
		require.NoError(t, err)

		// Push.
		_, _, err = git.NewCommand(git.DefaultContext, "push").AddArguments(git.ToTrustedCmdArgs([]string{"origin", "master"})...).RunStdString(&git.RunOpts{Dir: dstPath})
		require.NoError(t, err)

		// Check on page view.
		t.Run("Page view", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequest(t, http.MethodGet, "/user2/repo1/wiki/Home")
			resp := MakeRequest(t, req, http.StatusOK)
			htmlDoc := NewHTMLParser(t, resp.Body)

			htmlDoc.AssertElement(t, "script.evil", false)
			assert.Contains(t, htmlDoc.Find(".ui.sub.header").Text(), `Gusted<script class="evil">alert('Oh no!');</script> edited this page 2024-01-31`)
		})

		// Check on revisions page.
		t.Run("Revision page", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequest(t, http.MethodGet, "/user2/repo1/wiki/Home?action=_revision")
			resp := MakeRequest(t, req, http.StatusOK)
			htmlDoc := NewHTMLParser(t, resp.Body)

			htmlDoc.AssertElement(t, "script.evil", false)
			assert.Contains(t, htmlDoc.Find(".ui.sub.header").Text(), `Gusted<script class="evil">alert('Oh no!');</script> edited this page 2024-01-31`)
		})
	})
}

func TestXSSReviewDismissed(t *testing.T) {
	defer tests.AddFixtures("tests/integration/fixtures/TestXSSReviewDismissed/")()
	defer tests.PrepareTestEnv(t)()

	review := unittest.AssertExistsAndLoadBean(t, &issues_model.Review{ID: 1000})

	req := NewRequest(t, http.MethodGet, fmt.Sprintf("/user2/repo1/pulls/%d", +review.IssueID))
	resp := MakeRequest(t, req, http.StatusOK)
	htmlDoc := NewHTMLParser(t, resp.Body)

	htmlDoc.AssertElement(t, "script.evil", false)
	assert.Contains(t, htmlDoc.Find("#issuecomment-1000 .dismissed-message").Text(), `dismissed Otto <script class='evil'>alert('Oh no!')</script>'s review`)
}
