// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forms

import (
	"mime/multipart"
	"net/http"

	"code.gitea.io/gitea/modules/web/middleware"
	"code.gitea.io/gitea/services/context"

	"code.forgejo.org/go-chi/binding"
)

// NewBranchForm form for creating a new branch
type NewBranchForm struct {
	NewBranchName string `binding:"Required;MaxSize(100);GitRefName"`
	CurrentPath   string
	CreateTag     bool
}

// Validate validates the fields
func (f *NewBranchForm) Validate(req *http.Request, errs binding.Errors) binding.Errors {
	ctx := context.GetValidateContext(req)
	return middleware.Validate(errs, ctx.Data, f, ctx.Locale)
}

// RenameBranchForm form for rename a branch
type RenameBranchForm struct {
	From string `binding:"Required;MaxSize(100);GitRefName"`
	To   string `binding:"Required;MaxSize(100);GitRefName"`
}

// Validate validates the fields
func (f *RenameBranchForm) Validate(req *http.Request, errs binding.Errors) binding.Errors {
	ctx := context.GetValidateContext(req)
	return middleware.Validate(errs, ctx.Data, f, ctx.Locale)
}

type PackageUploadAlpineForm struct {
	Repo       string
	Branch     string
	Repository string
	File       *multipart.FileHeader
}

type PackageUploadDebianForm struct {
	Repo         string
	Distribution string
	Component    string
	File         *multipart.FileHeader
}

type PackageUploadGenericForm struct {
	Repo     string
	Name     string
	Version  string
	Filename string
	File     *multipart.FileHeader
}

type PackageUploadRpmForm struct {
	Repo  string
	Group string
	Sign  bool
	File  *multipart.FileHeader
}
