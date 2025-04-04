// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"io"
	"net/http"
	"net/url"
	"testing"

	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	"forgejo.org/modules/git"
	"forgejo.org/modules/gitrepo"
	"forgejo.org/modules/web"
	"forgejo.org/services/contexttest"
	"forgejo.org/services/forms"
	wiki_service "forgejo.org/services/wiki"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	content = "Wiki contents for unit tests"
	message = "Wiki commit message for unit tests"
)

func wikiEntry(t *testing.T, repo *repo_model.Repository, wikiName wiki_service.WebPath) *git.TreeEntry {
	wikiRepo, err := gitrepo.OpenWikiRepository(git.DefaultContext, repo)
	require.NoError(t, err)
	defer wikiRepo.Close()
	commit, err := wikiRepo.GetBranchCommit("master")
	require.NoError(t, err)
	entries, err := commit.ListEntries()
	require.NoError(t, err)
	for _, entry := range entries {
		if entry.Name() == wiki_service.WebPathToGitPath(wikiName) {
			return entry
		}
	}
	return nil
}

func wikiContent(t *testing.T, repo *repo_model.Repository, wikiName wiki_service.WebPath) string {
	entry := wikiEntry(t, repo, wikiName)
	if !assert.NotNil(t, entry) {
		return ""
	}
	reader, err := entry.Blob().DataAsync()
	require.NoError(t, err)
	defer reader.Close()
	bytes, err := io.ReadAll(reader)
	require.NoError(t, err)
	return string(bytes)
}

func assertWikiExists(t *testing.T, repo *repo_model.Repository, wikiName wiki_service.WebPath) {
	assert.NotNil(t, wikiEntry(t, repo, wikiName))
}

func assertWikiNotExists(t *testing.T, repo *repo_model.Repository, wikiName wiki_service.WebPath) {
	assert.Nil(t, wikiEntry(t, repo, wikiName))
}

func assertPagesMetas(t *testing.T, expectedNames []string, metas any) {
	pageMetas, ok := metas.([]PageMeta)
	if !assert.True(t, ok) {
		return
	}
	if !assert.Len(t, pageMetas, len(expectedNames)) {
		return
	}
	for i, pageMeta := range pageMetas {
		assert.Equal(t, expectedNames[i], pageMeta.Name)
	}
}

func TestWiki(t *testing.T) {
	unittest.PrepareTestEnv(t)

	ctx, _ := contexttest.MockContext(t, "user2/repo1/wiki")
	ctx.SetParams("*", "Home")
	contexttest.LoadRepo(t, ctx, 1)
	Wiki(ctx)
	assert.Equal(t, http.StatusOK, ctx.Resp.Status())
	assert.EqualValues(t, "Home", ctx.Data["Title"])
	assertPagesMetas(t, []string{"Home", "Long Page", "Page With Image", "Page With Spaced Name", "Unescaped File", "XSS"}, ctx.Data["Pages"])
}

func TestWikiPages(t *testing.T) {
	unittest.PrepareTestEnv(t)

	ctx, _ := contexttest.MockContext(t, "user2/repo1/wiki/?action=_pages")
	contexttest.LoadRepo(t, ctx, 1)
	WikiPages(ctx)
	assert.Equal(t, http.StatusOK, ctx.Resp.Status())
	assertPagesMetas(t, []string{"Home", "Long Page", "Page With Image", "Page With Spaced Name", "Unescaped File", "XSS"}, ctx.Data["Pages"])
}

func TestNewWiki(t *testing.T) {
	unittest.PrepareTestEnv(t)

	ctx, _ := contexttest.MockContext(t, "user2/repo1/wiki/?action=_new")
	contexttest.LoadUser(t, ctx, 2)
	contexttest.LoadRepo(t, ctx, 1)
	NewWiki(ctx)
	assert.Equal(t, http.StatusOK, ctx.Resp.Status())
	assert.EqualValues(t, ctx.Tr("repo.wiki.new_page"), ctx.Data["Title"])
}

func TestNewWikiPost(t *testing.T) {
	for _, title := range []string{
		"New page",
		"&&&&",
	} {
		unittest.PrepareTestEnv(t)

		ctx, _ := contexttest.MockContext(t, "user2/repo1/wiki/?action=_new")
		contexttest.LoadUser(t, ctx, 2)
		contexttest.LoadRepo(t, ctx, 1)
		web.SetForm(ctx, &forms.NewWikiForm{
			Title:   title,
			Content: content,
			Message: message,
		})
		NewWikiPost(ctx)
		assert.Equal(t, http.StatusSeeOther, ctx.Resp.Status())
		assertWikiExists(t, ctx.Repo.Repository, wiki_service.UserTitleToWebPath("", title))
		assert.Equal(t, content, wikiContent(t, ctx.Repo.Repository, wiki_service.UserTitleToWebPath("", title)))
	}
}

func TestNewWikiPost_ReservedName(t *testing.T) {
	unittest.PrepareTestEnv(t)

	ctx, _ := contexttest.MockContext(t, "user2/repo1/wiki/?action=_new")
	contexttest.LoadUser(t, ctx, 2)
	contexttest.LoadRepo(t, ctx, 1)
	web.SetForm(ctx, &forms.NewWikiForm{
		Title:   "_edit",
		Content: content,
		Message: message,
	})
	NewWikiPost(ctx)
	assert.Equal(t, http.StatusOK, ctx.Resp.Status())
	assert.EqualValues(t, ctx.Tr("repo.wiki.reserved_page"), ctx.Flash.ErrorMsg)
	assertWikiNotExists(t, ctx.Repo.Repository, "_edit")
}

func TestEditWiki(t *testing.T) {
	unittest.PrepareTestEnv(t)

	ctx, _ := contexttest.MockContext(t, "user2/repo1/wiki/Home?action=_edit")
	ctx.SetParams("*", "Home")
	contexttest.LoadUser(t, ctx, 2)
	contexttest.LoadRepo(t, ctx, 1)
	EditWiki(ctx)
	assert.Equal(t, http.StatusOK, ctx.Resp.Status())
	assert.EqualValues(t, "Home", ctx.Data["Title"])
	assert.Equal(t, wikiContent(t, ctx.Repo.Repository, "Home"), ctx.Data["content"])
}

func TestEditWikiPost(t *testing.T) {
	for _, title := range []string{
		"Home",
		"New/<page>",
	} {
		unittest.PrepareTestEnv(t)
		ctx, _ := contexttest.MockContext(t, "user2/repo1/wiki/Home?action=_new")
		ctx.SetParams("*", "Home")
		contexttest.LoadUser(t, ctx, 2)
		contexttest.LoadRepo(t, ctx, 1)
		web.SetForm(ctx, &forms.NewWikiForm{
			Title:   title,
			Content: content,
			Message: message,
		})
		EditWikiPost(ctx)
		assert.Equal(t, http.StatusSeeOther, ctx.Resp.Status())
		assertWikiExists(t, ctx.Repo.Repository, wiki_service.UserTitleToWebPath("", title))
		assert.Equal(t, content, wikiContent(t, ctx.Repo.Repository, wiki_service.UserTitleToWebPath("", title)))
		if title != "Home" {
			assertWikiNotExists(t, ctx.Repo.Repository, "Home")
		}
	}
}

func TestDeleteWikiPagePost(t *testing.T) {
	unittest.PrepareTestEnv(t)

	ctx, _ := contexttest.MockContext(t, "user2/repo1/wiki/Home?action=_delete")
	contexttest.LoadUser(t, ctx, 2)
	contexttest.LoadRepo(t, ctx, 1)
	DeleteWikiPagePost(ctx)
	assert.Equal(t, http.StatusOK, ctx.Resp.Status())
	assertWikiNotExists(t, ctx.Repo.Repository, "Home")
}

func TestWikiRaw(t *testing.T) {
	for filepath, filetype := range map[string]string{
		"jpeg.jpg":                 "image/jpeg",
		"images/jpeg.jpg":          "image/jpeg",
		"Page With Spaced Name":    "text/plain; charset=utf-8",
		"Page-With-Spaced-Name":    "text/plain; charset=utf-8",
		"Page With Spaced Name.md": "", // there is no "Page With Spaced Name.md" in repo
		"Page-With-Spaced-Name.md": "text/plain; charset=utf-8",
	} {
		unittest.PrepareTestEnv(t)

		ctx, _ := contexttest.MockContext(t, "user2/repo1/wiki/raw/"+url.PathEscape(filepath))
		ctx.SetParams("*", filepath)
		contexttest.LoadUser(t, ctx, 2)
		contexttest.LoadRepo(t, ctx, 1)
		WikiRaw(ctx)
		if filetype == "" {
			assert.Equal(t, http.StatusNotFound, ctx.Resp.Status(), "filepath: %s", filepath)
		} else {
			assert.Equal(t, http.StatusOK, ctx.Resp.Status(), "filepath: %s", filepath)
			assert.Equal(t, filetype, ctx.Resp.Header().Get("Content-Type"), "filepath: %s", filepath)
		}
	}
}
