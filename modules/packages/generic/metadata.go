// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package generic

import (
	"code.gitea.io/gitea/modules/util"
)

var (
	// ErrInvalidName indicates an invalid package name
	ErrInvalidName = util.NewInvalidArgumentErrorf("package name is invalid")
	// ErrInvalidFilename indicates an invalid filename
	ErrInvalidFilename = util.NewInvalidArgumentErrorf("invalid filename")
	// ErrInvalidVersion indicates an invalid package version
	ErrInvalidVersion = util.NewInvalidArgumentErrorf("package version is invalid")
)
