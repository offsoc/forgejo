// Copyright 2024 The Forgejo Authors. All rights reserved.
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
	"forgejo.org/routers/web/repo"
	"forgejo.org/services/context"
	"forgejo.org/services/contexttest"
	files_service "forgejo.org/services/repository/files"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
)

func createRepoAndGetContext(t *testing.T, files []string, deleteMdReadme bool) (*context.Context, func()) {
	t.Helper()

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: "user1"})

	size := len(files)
	if deleteMdReadme {
		size++
	}
	changeFiles := make([]*files_service.ChangeRepoFile, size)
	for i, e := range files {
		changeFiles[i] = &files_service.ChangeRepoFile{
			Operation:     "create",
			TreePath:      e,
			ContentReader: strings.NewReader("test"),
		}
	}
	if deleteMdReadme {
		changeFiles[len(files)] = &files_service.ChangeRepoFile{
			Operation: "delete",
			TreePath:  "README.md",
		}
	}

	// README.md is already added by auto init
	repo, _, f := tests.CreateDeclarativeRepo(t, user, "readmetest", []unit_model.Type{unit_model.TypeCode}, nil, changeFiles)

	ctx, _ := contexttest.MockContext(t, "user1/readmetest")
	ctx.SetParams(":id", fmt.Sprint(repo.ID))
	contexttest.LoadRepo(t, ctx, repo.ID)
	contexttest.LoadGitRepo(t, ctx)
	contexttest.LoadRepoCommit(t, ctx)

	return ctx, func() {
		f()
		ctx.Repo.GitRepo.Close()
	}
}

func TestRepoView_FindReadme(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		t.Run("PrioOneLocalizedMdReadme", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			ctx, f := createRepoAndGetContext(t, []string{"README.en.md", "README.en.org", "README.org", "README.txt", "README.tex"}, false)
			defer f()

			tree, _ := ctx.Repo.Commit.SubTree(ctx.Repo.TreePath)
			entries, _ := tree.ListEntries()
			_, file, _ := repo.FindReadmeFileInEntries(ctx, entries, false)

			assert.Equal(t, "README.en.md", file.Name())
		})
		t.Run("PrioTwoMdReadme", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			ctx, f := createRepoAndGetContext(t, []string{"README.en.org", "README.org", "README.txt", "README.tex"}, false)
			defer f()

			tree, _ := ctx.Repo.Commit.SubTree(ctx.Repo.TreePath)
			entries, _ := tree.ListEntries()
			_, file, _ := repo.FindReadmeFileInEntries(ctx, entries, false)

			assert.Equal(t, "README.md", file.Name())
		})
		t.Run("PrioThreeLocalizedOrgReadme", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			ctx, f := createRepoAndGetContext(t, []string{"README.en.org", "README.org", "README.txt", "README.tex"}, true)
			defer f()

			tree, _ := ctx.Repo.Commit.SubTree(ctx.Repo.TreePath)
			entries, _ := tree.ListEntries()
			_, file, _ := repo.FindReadmeFileInEntries(ctx, entries, false)

			assert.Equal(t, "README.en.org", file.Name())
		})
		t.Run("PrioFourOrgReadme", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			ctx, f := createRepoAndGetContext(t, []string{"README.org", "README.txt", "README.tex"}, true)
			defer f()

			tree, _ := ctx.Repo.Commit.SubTree(ctx.Repo.TreePath)
			entries, _ := tree.ListEntries()
			_, file, _ := repo.FindReadmeFileInEntries(ctx, entries, false)

			assert.Equal(t, "README.org", file.Name())
		})
		t.Run("PrioFiveTxtReadme", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			ctx, f := createRepoAndGetContext(t, []string{"README.txt", "README", "README.tex"}, true)
			defer f()

			tree, _ := ctx.Repo.Commit.SubTree(ctx.Repo.TreePath)
			entries, _ := tree.ListEntries()
			_, file, _ := repo.FindReadmeFileInEntries(ctx, entries, false)

			assert.Equal(t, "README.txt", file.Name())
		})
		t.Run("PrioSixWithoutExtensionReadme", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			ctx, f := createRepoAndGetContext(t, []string{"README", "README.tex"}, true)
			defer f()

			tree, _ := ctx.Repo.Commit.SubTree(ctx.Repo.TreePath)
			entries, _ := tree.ListEntries()
			_, file, _ := repo.FindReadmeFileInEntries(ctx, entries, false)

			assert.Equal(t, "README", file.Name())
		})
		t.Run("PrioSevenAnyReadme", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			ctx, f := createRepoAndGetContext(t, []string{"README.tex"}, true)
			defer f()

			tree, _ := ctx.Repo.Commit.SubTree(ctx.Repo.TreePath)
			entries, _ := tree.ListEntries()
			_, file, _ := repo.FindReadmeFileInEntries(ctx, entries, false)

			assert.Equal(t, "README.tex", file.Name())
		})
		t.Run("DoNotPickReadmeIfNonPresent", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			ctx, f := createRepoAndGetContext(t, []string{}, true)
			defer f()

			tree, _ := ctx.Repo.Commit.SubTree(ctx.Repo.TreePath)
			entries, _ := tree.ListEntries()
			_, file, _ := repo.FindReadmeFileInEntries(ctx, entries, false)

			assert.Nil(t, file)
		})
	})
}

func TestRepoViewFileLines(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, _ *url.URL) {
		user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
		repo, _, f := tests.CreateDeclarativeRepo(t, user, "file-lines", []unit_model.Type{unit_model.TypeCode}, nil, []*files_service.ChangeRepoFile{
			{
				Operation:     "create",
				TreePath:      "test-1",
				ContentReader: strings.NewReader("No newline"),
			},
			{
				Operation:     "create",
				TreePath:      "test-2",
				ContentReader: strings.NewReader("No newline\n"),
			},
			{
				Operation:     "create",
				TreePath:      "test-3",
				ContentReader: strings.NewReader("Two\nlines"),
			},
			{
				Operation:     "create",
				TreePath:      "test-4",
				ContentReader: strings.NewReader("Really two\nlines\n"),
			},
			{
				Operation:     "create",
				TreePath:      "empty",
				ContentReader: strings.NewReader(""),
			},
			{
				Operation:     "create",
				TreePath:      "seemingly-empty",
				ContentReader: strings.NewReader("\n"),
			},
		})
		defer f()

		testEOL := func(t *testing.T, filename string, hasEOL bool) {
			t.Helper()
			req := NewRequestf(t, "GET", "%s/src/branch/main/%s", repo.Link(), filename)
			resp := MakeRequest(t, req, http.StatusOK)
			htmlDoc := NewHTMLParser(t, resp.Body)

			fileInfo := htmlDoc.Find(".file-info").Text()
			if hasEOL {
				assert.NotContains(t, fileInfo, "No EOL")
			} else {
				assert.Contains(t, fileInfo, "No EOL")
			}
		}

		t.Run("No EOL", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			testEOL(t, "test-1", false)
			testEOL(t, "test-3", false)
		})

		t.Run("With EOL", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			testEOL(t, "test-2", true)
			testEOL(t, "test-4", true)
			testEOL(t, "empty", true)
			testEOL(t, "seemingly-empty", true)
		})
	})
}
