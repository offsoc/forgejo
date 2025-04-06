// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package internal

import "forgejo.org/modules/indexer/internal"

func FilenameIndexerID(repoID int64, filename string) string {
	return internal.Base36(repoID) + "_" + filename
}
