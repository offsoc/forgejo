// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/optional"
	"code.gitea.io/gitea/modules/util"
	wiki_service "code.gitea.io/gitea/services/wiki"
	"code.gitea.io/gitea/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func assertFileExist(t *testing.T, p string) {
	exist, err := util.IsExist(p)
	require.NoError(t, err)
	assert.True(t, exist)
}

func assertFileEqual(t *testing.T, p string, content []byte) {
	bs, err := os.ReadFile(p)
	require.NoError(t, err)
	assert.EqualValues(t, content, bs)
}

func TestRepoCloneWiki(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		dstPath := t.TempDir()

		r := fmt.Sprintf("%suser2/repo1.wiki.git", u.String())
		u, _ = url.Parse(r)
		u.User = url.UserPassword("user2", userPassword)
		t.Run("Clone", func(t *testing.T) {
			require.NoError(t, git.CloneWithArgs(context.Background(), git.AllowLFSFiltersArgs(), u.String(), dstPath, git.CloneRepoOptions{}))
			assertFileEqual(t, filepath.Join(dstPath, "Home.md"), []byte("# Home page\n\nThis is the home page!\n"))
			assertFileExist(t, filepath.Join(dstPath, "Page-With-Image.md"))
			assertFileExist(t, filepath.Join(dstPath, "Page-With-Spaced-Name.md"))
			assertFileExist(t, filepath.Join(dstPath, "images"))
			assertFileExist(t, filepath.Join(dstPath, "jpeg.jpg"))
		})
	})
}

func Test_RepoWikiPages(t *testing.T) {
	userName := "user1"
	repoName := "some-repo"
	repoPath := userName + "/" + repoName
	wikiPath := "/" + repoPath + "/wiki/"
	wikiPages := []struct {
		createPath string
		expectPath string
	}{
		{"Home", "Home"},
		{"_Sidebar", "_Sidebar"},
		{"small", "small"},
		{"snake_scary", "snake_scary"},
		{"ke-bab", "ke-bab"},
		{"Spaced Page", "Spaced Page"},
		{"Page%AllPages", "Page%AllPages"},
		{"Cake/Lie", "Cake/Lie"},
	}
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		// Prep
		user := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: userName})

		repo, _, f := tests.CreateDeclarativeRepoWithOptions(t, user, tests.DeclarativeRepoOptions{
			Name:       optional.Some(repoName),
			WikiBranch: optional.Some("master"),
		})
		defer f()
		err := wiki_service.DeleteWikiPage(db.DefaultContext, user, repo, "Home")
		require.NoError(t, err, "unable to clean wiki to be empty")

		for _, page := range wikiPages {
			err := wiki_service.AddWikiPage(
				db.DefaultContext,
				user,
				repo,
				wiki_service.WebPath(page.createPath),
				"",
				"",
			)
			require.NoError(t, err, "could't create wiki page")

			// Test
			req := NewRequest(t, "GET", wikiPath+"?action=_pages")
			resp := MakeRequest(t, req, http.StatusOK)

			doc := NewHTMLParser(t, resp.Body)
			s := doc.Find("table.wiki-pages-list>tbody>tr>td").First()
			anchor := s.Find("a").First()

			text := anchor.Text()
			assert.EqualValues(t, page.expectPath, text)

			href, exists := anchor.Attr("href")
			assert.True(t, exists)
			href = strings.TrimPrefix(href, wikiPath)
			href, err = url.PathUnescape(href)
			require.NoError(t, err)
			assert.EqualValues(t, page.expectPath, href)

			// Cleanup
			err = wiki_service.DeleteWikiPage(
				db.DefaultContext,
				user,
				repo,
				wiki_service.WebPath(page.expectPath),
			)
			require.NoError(t, err, "unable to cleanup page for next case")
		}
	})
}
