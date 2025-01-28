// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package unittest

import (
	"os"
)

// CopyDir copy files recursively from source to target directory.
//
// It returns error when error occurs in underlying functions.
func CopyDir(srcPath, destPath string) error {
	return os.CopyFS(destPath, os.DirFS(srcPath))
}
