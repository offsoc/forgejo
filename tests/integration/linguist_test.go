// Copyright 2024 The Forgejo Authors c/o Codeberg e.V.. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"code.gitea.io/gitea/models/db"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/indexer/stats"
	"code.gitea.io/gitea/modules/queue"
	files_service "code.gitea.io/gitea/services/repository/files"
	"code.gitea.io/gitea/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLinguistSupport(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		/******************
		 ** Preparations **
		 ******************/
		prep := func(t *testing.T, attribs string) (*repo_model.Repository, string, func()) {
			t.Helper()

			user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

			repo, sha, f := tests.CreateDeclarativeRepo(t, user2, "", nil, nil,
				[]*files_service.ChangeRepoFile{
					{
						Operation:     "create",
						TreePath:      ".gitattributes",
						ContentReader: strings.NewReader(attribs),
					},
					{
						Operation:     "create",
						TreePath:      "docs.md",
						ContentReader: strings.NewReader("This **is** a `markdown` file.\n"),
					},
					{
						Operation:     "create",
						TreePath:      "foo.c",
						ContentReader: strings.NewReader("#include <stdio.h>\nint main() {\n  printf(\"Hello world!\n\");\n  return 0;\n}\n"),
					},
					{
						Operation:     "create",
						TreePath:      "foo.nib",
						ContentReader: strings.NewReader("Pinky promise, this is not a generated file!\n"),
					},
					{
						Operation:     "create",
						TreePath:      ".dot.pas",
						ContentReader: strings.NewReader("program Hello;\nbegin\n  writeln('Hello, world.');\nend.\n"),
					},
					{
						Operation:     "create",
						TreePath:      "cpplint.py",
						ContentReader: strings.NewReader("#! /usr/bin/env python\n\nprint(\"Hello world!\")\n"),
					},
					{
						Operation:     "create",
						TreePath:      "some-file.xml",
						ContentReader: strings.NewReader("<?xml version=\"1.0\"?>\n<foo>\n <bar>Hello</bar>\n</foo>\n"),
					},
				})

			return repo, sha, f
		}

		getFreshLanguageStats := func(t *testing.T, repo *repo_model.Repository, sha string) repo_model.LanguageStatList {
			t.Helper()

			err := stats.UpdateRepoIndexer(repo)
			require.NoError(t, err)

			require.NoError(t, queue.GetManager().FlushAll(t.Context(), 10*time.Second))

			status, err := repo_model.GetIndexerStatus(db.DefaultContext, repo, repo_model.RepoIndexerTypeStats)
			require.NoError(t, err)
			assert.Equal(t, sha, status.CommitSha)
			langs, err := repo_model.GetTopLanguageStats(db.DefaultContext, repo, 5)
			require.NoError(t, err)

			return langs
		}

		/***********
		 ** Tests **
		 ***********/

		// 1. By default, documentation is not indexed
		t.Run("default", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			repo, sha, f := prep(t, "")
			defer f()

			langs := getFreshLanguageStats(t, repo, sha)

			// While this is a fairly short test, this exercises a number of
			// things:
			//
			// - `.gitattributes` is empty, so `isDetectable.IsFalse()`,
			//   `isVendored.IsTrue()`, and `isDocumentation.IsTrue()` will be
			//   false for every file, because these are only true if an
			//   attribute is explicitly set.
			//
			// - There is `.dot.pas`, which would be considered Pascal source,
			//   but it is a dotfile (thus, `enry.IsDotFile()` applies), and as
			//   such, is not considered.
			//
			// - `some-file.xml` will be skipped because Enry considers XML
			//   configuration, and `enry.IsConfiguration()` will catch it.
			//
			// - `!isVendored.IsFalse()` evaluates to true, so
			//   `analyze.isVendor()` will be called on `cpplint.py`, which will
			//   be considered vendored, even though both the filename and
			//   contents would otherwise make it Python.
			//
			// - `!isDocumentation.IsFalse()` evaluates to true, so
			//   `enry.IsDocumentation()` will be called for `docs.md`, and will
			//   be considered documentation, thus, skipped.
			//
			// Thus, this exercises all of the conditions in the first big if
			// that is supposed to filter out files early. With two short asserts!

			assert.Len(t, langs, 1)
			assert.Equal(t, "C", langs[0].Language)
		})

		// 2. Marking foo.c as non-detectable
		t.Run("foo.c non-detectable", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			repo, sha, f := prep(t, "foo.c linguist-detectable=false\n")
			defer f()

			langs := getFreshLanguageStats(t, repo, sha)
			assert.Empty(t, langs)
		})

		// 3. Marking Markdown detectable
		t.Run("detectable markdown", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			repo, sha, f := prep(t, "*.md linguist-detectable\n")
			defer f()

			langs := getFreshLanguageStats(t, repo, sha)
			assert.Len(t, langs, 2)
			assert.Equal(t, "C", langs[0].Language)
			assert.Equal(t, "Markdown", langs[1].Language)
		})

		// 4. Marking foo.c as documentation
		t.Run("foo.c as documentation", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			repo, sha, f := prep(t, "foo.c linguist-documentation\n")
			defer f()

			langs := getFreshLanguageStats(t, repo, sha)
			assert.Empty(t, langs)
		})

		// 5. Overriding a generated file
		t.Run("linguist-generated=false", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			repo, sha, f := prep(t, "foo.nib linguist-generated=false\nfoo.nib linguist-language=Perl\n")
			defer f()

			langs := getFreshLanguageStats(t, repo, sha)
			assert.Len(t, langs, 2)
			assert.Equal(t, "C", langs[0].Language)
			assert.Equal(t, "Perl", langs[1].Language)
		})

		// 6. Disabling vendoring for a file
		t.Run("linguist-vendored=false", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			repo, sha, f := prep(t, "cpplint.py linguist-vendored=false\n")
			defer f()

			langs := getFreshLanguageStats(t, repo, sha)
			assert.Len(t, langs, 2)
			assert.Equal(t, "C", langs[0].Language)
			assert.Equal(t, "Python", langs[1].Language)
		})

		// 7. Disabling vendoring for a file, with -linguist-vendored
		t.Run("-linguist-vendored", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			repo, sha, f := prep(t, "cpplint.py -linguist-vendored\n")
			defer f()

			langs := getFreshLanguageStats(t, repo, sha)
			assert.Len(t, langs, 2)
			assert.Equal(t, "C", langs[0].Language)
			assert.Equal(t, "Python", langs[1].Language)
		})

		// 8. Marking foo.c as vendored
		t.Run("foo.c as vendored", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			repo, sha, f := prep(t, "foo.c linguist-vendored\n")
			defer f()

			langs := getFreshLanguageStats(t, repo, sha)
			assert.Empty(t, langs)
		})

		// 9. Overriding the language
		t.Run("linguist-language", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			repo, _, f := prep(t, "foo.c linguist-language=sh\n")
			defer f()

			assertFileLanguage := func(t *testing.T, uri, expectedLanguage string) {
				t.Helper()

				req := NewRequest(t, "GET", repo.Link()+uri)
				resp := MakeRequest(t, req, http.StatusOK)
				htmlDoc := NewHTMLParser(t, resp.Body)

				language := strings.TrimSpace(htmlDoc.Find(".file-info .file-info-entry:nth-child(3)").Text())
				assert.Equal(t, expectedLanguage, language)
			}

			t.Run("file source view", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				assertFileLanguage(t, "/src/branch/main/foo.c?display=source", "Bash")
			})

			t.Run("file blame view", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				assertFileLanguage(t, "/blame/branch/main/foo.c", "Bash")
			})
		})

		// 10. Marking a file as non-documentation
		t.Run("linguist-documentation=false", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			repo, sha, f := prep(t, "README.md linguist-documentation=false\n")
			defer f()

			langs := getFreshLanguageStats(t, repo, sha)
			assert.Len(t, langs, 2)
			assert.Equal(t, "Markdown", langs[0].Language)
			assert.Equal(t, "C", langs[1].Language)
		})
	})
}
