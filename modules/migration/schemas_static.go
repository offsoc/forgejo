// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

//go:build bindata

package migration

import (
	"path"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

type SchemaLoader struct{}

func (*SchemaLoader) Load(filename string) (any, error) {
	f, err := Assets.Open(path.Base(filename))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return jsonschema.UnmarshalJSON(f)
}
