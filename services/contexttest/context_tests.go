// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

// Package contexttest provides utilities for testing Web/API contexts with models.
package contexttest

import (
	gocontext "context"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	org_model "forgejo.org/models/organization"
	access_model "forgejo.org/models/perm/access"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/gitrepo"
	"forgejo.org/modules/templates"
	"forgejo.org/modules/translation"
	"forgejo.org/modules/web/middleware"
	"forgejo.org/services/context"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mockRequest(t *testing.T, reqPath string) *http.Request {
	method, path, found := strings.Cut(reqPath, " ")
	if !found {
		method = "GET"
		path = reqPath
	}
	requestURL, err := url.Parse(path)
	require.NoError(t, err)
	req := &http.Request{Method: method, URL: requestURL, Form: maps.Clone(requestURL.Query()), Header: http.Header{}}
	req = req.WithContext(middleware.WithContextData(req.Context()))
	return req
}

type MockContextOption struct {
	Render context.Render
}

// MockContext mock context for unit tests
func MockContext(t *testing.T, reqPath string, opts ...MockContextOption) (*context.Context, *httptest.ResponseRecorder) {
	var opt MockContextOption
	if len(opts) > 0 {
		opt = opts[0]
	}
	if opt.Render == nil {
		opt.Render = &MockRender{}
	}
	resp := httptest.NewRecorder()
	req := mockRequest(t, reqPath)
	base, baseCleanUp := context.NewBaseContext(resp, req)
	_ = baseCleanUp // during test, it doesn't need to do clean up. TODO: this can be improved later
	base.Data = middleware.GetContextData(req.Context())
	base.Locale = &translation.MockLocale{}

	ctx := context.NewWebContext(base, opt.Render, nil)
	ctx.PageData = map[string]any{}
	ctx.Data["PageStartTime"] = time.Now()
	chiCtx := chi.NewRouteContext()
	ctx.AppendContextValue(chi.RouteCtxKey, chiCtx)
	return ctx, resp
}

// MockAPIContext mock context for unit tests
func MockAPIContext(t *testing.T, reqPath string) (*context.APIContext, *httptest.ResponseRecorder) {
	resp := httptest.NewRecorder()
	req := mockRequest(t, reqPath)
	base, baseCleanUp := context.NewBaseContext(resp, req)
	base.Data = middleware.GetContextData(req.Context())
	base.Locale = &translation.MockLocale{}
	ctx := &context.APIContext{Base: base}
	_ = baseCleanUp // during test, it doesn't need to do clean up. TODO: this can be improved later

	chiCtx := chi.NewRouteContext()
	ctx.AppendContextValue(chi.RouteCtxKey, chiCtx)
	return ctx, resp
}

func MockPrivateContext(t *testing.T, reqPath string) (*context.PrivateContext, *httptest.ResponseRecorder) {
	resp := httptest.NewRecorder()
	req := mockRequest(t, reqPath)
	base, baseCleanUp := context.NewBaseContext(resp, req)
	base.Data = middleware.GetContextData(req.Context())
	base.Locale = &translation.MockLocale{}
	ctx := &context.PrivateContext{Base: base}
	_ = baseCleanUp // during test, it doesn't need to do clean up. TODO: this can be improved later
	chiCtx := chi.NewRouteContext()
	ctx.AppendContextValue(chi.RouteCtxKey, chiCtx)
	return ctx, resp
}

// LoadRepo load a repo into a test context.
func LoadRepo(t *testing.T, ctx gocontext.Context, repoID int64) {
	var doer *user_model.User
	repo := &context.Repository{}
	switch ctx := ctx.(type) {
	case *context.Context:
		ctx.Repo = repo
		doer = ctx.Doer
	case *context.APIContext:
		ctx.Repo = repo
		doer = ctx.Doer
	default:
		assert.FailNow(t, "context is not *context.Context or *context.APIContext")
	}

	repo.Repository = unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: repoID})
	var err error
	repo.Owner, err = user_model.GetUserByID(ctx, repo.Repository.OwnerID)
	require.NoError(t, err)
	repo.RepoLink = repo.Repository.Link()
	repo.Permission, err = access_model.GetUserRepoPermission(ctx, repo.Repository, doer)
	require.NoError(t, err)
}

// LoadRepoCommit loads a repo's commit into a test context.
func LoadRepoCommit(t *testing.T, ctx gocontext.Context) {
	var repo *context.Repository
	switch ctx := ctx.(type) {
	case *context.Context:
		repo = ctx.Repo
	case *context.APIContext:
		repo = ctx.Repo
	default:
		assert.FailNow(t, "context is not *context.Context or *context.APIContext")
	}

	if repo.GitRepo == nil {
		assert.FailNow(t, "must call LoadGitRepo")
	}

	branch, err := repo.GitRepo.GetHEADBranch()
	require.NoError(t, err)
	assert.NotNil(t, branch)
	if branch != nil {
		repo.Commit, err = repo.GitRepo.GetBranchCommit(branch.Name)
		require.NoError(t, err)
	}
}

// LoadUser load a user into a test context
func LoadUser(t *testing.T, ctx gocontext.Context, userID int64) {
	doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: userID})
	switch ctx := ctx.(type) {
	case *context.Context:
		ctx.Doer = doer
	case *context.APIContext:
		ctx.Doer = doer
	default:
		assert.FailNow(t, "context is not *context.Context or *context.APIContext")
	}
}

// LoadOrganization load an org into a test context
func LoadOrganization(t *testing.T, ctx gocontext.Context, orgID int64) {
	org := unittest.AssertExistsAndLoadBean(t, &org_model.Organization{ID: orgID})
	switch ctx := ctx.(type) {
	case *context.Context:
		ctx.Org.Organization = org
	case *context.APIContext:
		ctx.Org.Organization = org
	default:
		assert.FailNow(t, "context is not *context.Context or *context.APIContext")
	}
}

// LoadGitRepo load a git repo into a test context. Requires that ctx.Repo has
// already been populated.
func LoadGitRepo(t *testing.T, ctx gocontext.Context) {
	var repo *context.Repository
	switch ctx := ctx.(type) {
	case *context.Context:
		repo = ctx.Repo
	case *context.APIContext:
		repo = ctx.Repo
	default:
		assert.FailNow(t, "context is not *context.Context or *context.APIContext")
	}

	require.NoError(t, repo.Repository.LoadOwner(ctx))
	var err error
	repo.GitRepo, err = gitrepo.OpenRepository(ctx, repo.Repository)
	require.NoError(t, err)
}

type MockRender struct{}

func (tr *MockRender) TemplateLookup(tmpl string, _ gocontext.Context) (templates.TemplateExecutor, error) {
	return nil, nil
}

func (tr *MockRender) HTML(w io.Writer, status int, _ string, _ any, _ gocontext.Context) error {
	if resp, ok := w.(http.ResponseWriter); ok {
		resp.WriteHeader(status)
	}
	return nil
}
