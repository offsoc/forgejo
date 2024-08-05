// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	unit_model "code.gitea.io/gitea/models/unit"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/routers/web/repo"
	"code.gitea.io/gitea/services/context"
	"code.gitea.io/gitea/services/contexttest"
	files_service "code.gitea.io/gitea/services/repository/files"
	"code.gitea.io/gitea/tests"

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
	repo, _, f := CreateDeclarativeRepo(t, user, "readmetest", []unit_model.Type{unit_model.TypeCode}, nil, changeFiles)

	ctx, _ := contexttest.MockContext(t, "user1/readmetest")
	ctx.SetParams(":id", fmt.Sprint(repo.ID))
	contexttest.LoadRepo(t, ctx, repo.ID)
	contexttest.LoadRepoCommit(t, ctx)
	return ctx, f
}

func TestRepoView_FindReadme(t *testing.T) {
	t.Run("PrioOneLocalizedMdReadme", func(t *testing.T) {
		onGiteaRun(t, func(t *testing.T, u *url.URL) {
			ctx, f := createRepoAndGetContext(t, []string{"README.en.md", "README.en.org", "README.org", "README.txt", "README.tex"}, false)
			defer f()

			tree, _ := ctx.Repo.Commit.SubTree(ctx.Repo.TreePath)
			entries, _ := tree.ListEntries()
			_, file, _ := repo.FindReadmeFileInEntries(ctx, entries, false)

			assert.Equal(t, "README.en.md", file.Name())
		})
	})
	t.Run("PrioTwoMdReadme", func(t *testing.T) {
		onGiteaRun(t, func(t *testing.T, u *url.URL) {
			ctx, f := createRepoAndGetContext(t, []string{"README.en.org", "README.org", "README.txt", "README.tex"}, false)
			defer f()

			tree, _ := ctx.Repo.Commit.SubTree(ctx.Repo.TreePath)
			entries, _ := tree.ListEntries()
			_, file, _ := repo.FindReadmeFileInEntries(ctx, entries, false)

			assert.Equal(t, "README.md", file.Name())
		})
	})
	t.Run("PrioThreeLocalizedOrgReadme", func(t *testing.T) {
		onGiteaRun(t, func(t *testing.T, u *url.URL) {
			ctx, f := createRepoAndGetContext(t, []string{"README.en.org", "README.org", "README.txt", "README.tex"}, true)
			defer f()

			tree, _ := ctx.Repo.Commit.SubTree(ctx.Repo.TreePath)
			entries, _ := tree.ListEntries()
			_, file, _ := repo.FindReadmeFileInEntries(ctx, entries, false)

			assert.Equal(t, "README.en.org", file.Name())
		})
	})
	t.Run("PrioFourOrgReadme", func(t *testing.T) {
		onGiteaRun(t, func(t *testing.T, u *url.URL) {
			ctx, f := createRepoAndGetContext(t, []string{"README.org", "README.txt", "README.tex"}, true)
			defer f()

			tree, _ := ctx.Repo.Commit.SubTree(ctx.Repo.TreePath)
			entries, _ := tree.ListEntries()
			_, file, _ := repo.FindReadmeFileInEntries(ctx, entries, false)

			assert.Equal(t, "README.org", file.Name())
		})
	})
	t.Run("PrioFiveTxtReadme", func(t *testing.T) {
		onGiteaRun(t, func(t *testing.T, u *url.URL) {
			ctx, f := createRepoAndGetContext(t, []string{"README.txt", "README", "README.tex"}, true)
			defer f()

			tree, _ := ctx.Repo.Commit.SubTree(ctx.Repo.TreePath)
			entries, _ := tree.ListEntries()
			_, file, _ := repo.FindReadmeFileInEntries(ctx, entries, false)

			assert.Equal(t, "README.txt", file.Name())
		})
	})
	t.Run("PrioSixWithoutExtensionReadme", func(t *testing.T) {
		onGiteaRun(t, func(t *testing.T, u *url.URL) {
			ctx, f := createRepoAndGetContext(t, []string{"README", "README.tex"}, true)
			defer f()

			tree, _ := ctx.Repo.Commit.SubTree(ctx.Repo.TreePath)
			entries, _ := tree.ListEntries()
			_, file, _ := repo.FindReadmeFileInEntries(ctx, entries, false)

			assert.Equal(t, "README", file.Name())
		})
	})
	t.Run("PrioSevenAnyReadme", func(t *testing.T) {
		onGiteaRun(t, func(t *testing.T, u *url.URL) {
			ctx, f := createRepoAndGetContext(t, []string{"README.tex"}, true)
			defer f()

			tree, _ := ctx.Repo.Commit.SubTree(ctx.Repo.TreePath)
			entries, _ := tree.ListEntries()
			_, file, _ := repo.FindReadmeFileInEntries(ctx, entries, false)

			assert.Equal(t, "README.tex", file.Name())
		})
	})
	t.Run("DoNotPickReadmeIfNonPresent", func(t *testing.T) {
		onGiteaRun(t, func(t *testing.T, u *url.URL) {
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
		repo, _, f := CreateDeclarativeRepo(t, user, "file-lines", []unit_model.Type{unit_model.TypeCode}, nil, []*files_service.ChangeRepoFile{
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
		})
		defer f()

		t.Run("No EOL", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequest(t, "GET", repo.Link()+"/src/branch/main/test-1")
			resp := MakeRequest(t, req, http.StatusOK)
			htmlDoc := NewHTMLParser(t, resp.Body)

			fileInfo := htmlDoc.Find(".file-info").Text()
			assert.Contains(t, fileInfo, "No EOL")

			req = NewRequest(t, "GET", repo.Link()+"/src/branch/main/test-3")
			resp = MakeRequest(t, req, http.StatusOK)
			htmlDoc = NewHTMLParser(t, resp.Body)

			fileInfo = htmlDoc.Find(".file-info").Text()
			assert.Contains(t, fileInfo, "No EOL")
		})

		t.Run("With EOL", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequest(t, "GET", repo.Link()+"/src/branch/main/test-2")
			resp := MakeRequest(t, req, http.StatusOK)
			htmlDoc := NewHTMLParser(t, resp.Body)

			fileInfo := htmlDoc.Find(".file-info").Text()
			assert.NotContains(t, fileInfo, "No EOL")

			req = NewRequest(t, "GET", repo.Link()+"/src/branch/main/test-4")
			resp = MakeRequest(t, req, http.StatusOK)
			htmlDoc = NewHTMLParser(t, resp.Body)

			fileInfo = htmlDoc.Find(".file-info").Text()
			assert.NotContains(t, fileInfo, "No EOL")
		})
	})
}
