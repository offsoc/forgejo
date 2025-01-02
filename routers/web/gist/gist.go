// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package gist

import (
	"fmt"
	"net/http"
	"strings"

	gist_model "code.gitea.io/gitea/models/gist"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/sitemap"
	api "code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/routers/common"
	"code.gitea.io/gitea/services/context"
	gist_service "code.gitea.io/gitea/services/gist"
)

type gistForm struct {
	Name        string
	Description string
	Visibility  gist_model.GistVisibility
	Files       map[string]string
}

// parseGistForm parses the form
// This is needed, as the normal parser can't handle the multiple files
func parseGistForm(req *http.Request) (*gistForm, error) {
	err := req.ParseForm()
	if err != nil {
		return nil, err
	}

	form := new(gistForm)

	form.Name = req.FormValue("name")
	if form.Name == "" {
		return nil, fmt.Errorf("name can't be empty")
	}

	form.Description = req.FormValue("description")

	form.Visibility, err = gist_model.GistVisibilityFromName(req.FormValue("visibility"))
	if err != nil {
		return nil, err
	}

	form.Files = make(map[string]string)

	for key, value := range req.Form {
		if !strings.HasPrefix(key, "file-name-") {
			continue
		}

		if len(value) == 0 {
			return nil, fmt.Errorf("%s has no value", key)
		}

		name := value[0]

		fileID := strings.TrimPrefix(key, "file-name-")

		content := req.FormValue(fmt.Sprintf("file-content-%s", fileID))

		form.Files[name] = content
	}

	if len(form.Files) == 0 {
		return nil, fmt.Errorf("form has no files")
	}

	return form, nil
}

// New creates a Gist
func New(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("gist.edit.new_header")

	ctx.HTML(http.StatusOK, "gist/add_edit")
}

// NewPost handels the post event for a Gist page
func NewPost(ctx *context.Context) {
	form, err := parseGistForm(ctx.Req)
	if err != nil {
		ctx.ServerError("ParseGistForm", err)
		return
	}

	gist, err := gist_service.CreateGist(ctx, ctx.Doer, form.Name, form.Description, form.Visibility, form.Files)
	if err != nil {
		ctx.ServerError("CreateGist", err)
		return
	}

	ctx.Redirect(gist.Link())
}

// View shows a Gist
func View(ctx *context.Context) {
	err := ctx.Gist.LoadOwner(ctx)
	if err != nil {
		ctx.ServerError("LoadOwner", err)
		return
	}

	files, err := gist_service.GetFiles(ctx, ctx.Gist)
	if err != nil {
		ctx.ServerError("GetFiles", err)
		return
	}

	err = files.Highlight()
	if err != nil {
		ctx.ServerError("HighlightFiles", err)
		return
	}

	if ctx.Gist.Visibility == gist_model.GistVisibilityHidden {
		ctx.Data["AddNoIndexHeader"] = true
	}

	cl := new(repo_model.CloneLink)
	cl.SSH = repo_model.ComposeSSHCloneURL("gists", ctx.Gist.UUID)
	cl.HTTPS = repo_model.ComposeHTTPSCloneURL("gists", ctx.Gist.UUID)

	ctx.Data["RepoCloneLink"] = cl

	cloneButtonShowHTTPS := !setting.Repository.DisableHTTPGit
	cloneButtonShowSSH := !setting.SSH.Disabled && (ctx.IsSigned || setting.SSH.ExposeAnonymous)
	if !cloneButtonShowHTTPS && !cloneButtonShowSSH {
		// We have to show at least one link, so we just show the HTTPS
		cloneButtonShowHTTPS = true
	}
	ctx.Data["CloneButtonShowHTTPS"] = cloneButtonShowHTTPS
	ctx.Data["CloneButtonShowSSH"] = cloneButtonShowSSH
	ctx.Data["CloneButtonOriginLink"] = ctx.Data["RepoCloneLink"]

	ctx.Data["Gist"] = ctx.Gist
	ctx.Data["GistFiles"] = files
	ctx.Data["Title"] = ctx.Gist.Name

	ctx.HTML(http.StatusOK, "gist/view")
}

func Raw(ctx *context.Context) {
	filename := ctx.Params(":filename")

	blob, err := gist_service.GetBlob(ctx, ctx.Gist, filename)
	if err != nil {
		ctx.ServerError("GetBlob", err)
		return
	}

	err = common.ServeBlob(ctx.Base, filename, blob, nil)
	if err != nil {
		ctx.ServerError("ServeBlob", err)
		return
	}
}

// Edit show the edit page
func Edit(ctx *context.Context) {
	files, err := gist_service.GetFiles(ctx, ctx.Gist)
	if err != nil {
		ctx.ServerError("GetFiles", err)
		return
	}

	ctx.Data["Gist"] = ctx.Gist
	ctx.Data["GistFiles"] = files
	ctx.Data["Title"] = ctx.Tr("gist.edit.edit_header")

	ctx.HTML(http.StatusOK, "gist/add_edit")
}

// EditPost handels the post for the edit page
func EditPost(ctx *context.Context) {
	form, err := parseGistForm(ctx.Req)
	if err != nil {
		ctx.ServerError("ParseGistForm", err)
		return
	}

	ctx.Gist.Name = form.Name
	ctx.Gist.Description = form.Description
	ctx.Gist.Visibility = form.Visibility

	err = ctx.Gist.UpdateCols(ctx, "name", "description", "visibility")
	if err != nil {
		ctx.ServerError("UpdateCols", err)
		return
	}

	files := make(gist_service.GistFiles, 0)
	for name, content := range form.Files {
		files = append(files, &api.GistFile{Name: name, Content: content})
	}

	err = gist_service.UpdateFiles(ctx, ctx.Gist, ctx.Doer, files)
	if err != nil {
		ctx.ServerError("UpdateFiles", err)
		return
	}

	ctx.Redirect(ctx.Gist.Link())
}

// Delete deletes a Gist
func Delete(ctx *context.Context) {
	err := gist_service.DeleteGist(ctx, ctx.Gist)
	if err != nil {
		ctx.ServerError("DeleteGist", err)
		return
	}

	ctx.Flash.Success(ctx.Tr("gist.delete.success"))

	ctx.Redirect("/")
}

// Sitemap renders the Gists Sitemap
func Sitemap(ctx *context.Context) {
	opts := new(gist_model.SearchGistOptions)
	opts.Page = int(ctx.ParamsInt64("idx"))
	opts.PageSize = setting.UI.SitemapPagingNum

	gists, _, err := gist_model.SearchGist(ctx, opts)
	if err != nil {
		log.Error("Failed to get Gists: %v", err)
		return
	}

	m := sitemap.NewSitemap()
	for _, item := range gists {
		m.Add(sitemap.URL{URL: item.HTMLURL(), LastMod: item.UpdatedUnix.AsTimePtr()})
	}

	ctx.Resp.Header().Set("Content-Type", "text/xml")

	if _, err := m.WriteTo(ctx.Resp); err != nil {
		log.Error("Failed writing sitemap: %v", err)
	}
}
