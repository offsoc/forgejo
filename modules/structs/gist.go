// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package structs

import (
	"html/template"
	"time"
)

type Gist struct {
	ID          int64  `json:"id"`
	Owner       *User  `json:"owner"`
	UUID        string `json:"uuid"`
	Name        string `json:"name"`
	Description string `json:"description"`
	// enum: public,hidden,private
	Visibility string `json:"visibility"`
	// swagger:strfmt date-time
	Created time.Time `json:"created_at"`
	// swagger:strfmt date-time
	Updated time.Time `json:"updated_at"`
}

type GistList struct {
	Gists []*Gist `json:"gists"`
}

type CreateGistOption struct {
	Name string `json:"name" binding:"Required;MaxSize(100)"`
	// enum: public,hidden,private
	Visibility  string      `json:"visibility" binding:"Required"`
	Description string      `json:"description"`
	Files       []*GistFile `json:"files" binding:"Required"`
}

type GistFile struct {
	Name               string          `json:"name"`
	Content            string          `json:"content"`
	HighlightedContent []template.HTML `json:"-"`
}

type UpdateGistFilesOption struct {
	Files []*GistFile `json:"files"`
}
