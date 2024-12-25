// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package git

import (
	"context"
	"fmt"
)

// WriteCommitGraph write commit graph to speed up repo access
// this requires git v2.18 to be installed and optionally
// git v2.27 to enable bloom filters
func WriteCommitGraph(ctx context.Context, repoPath string) error {
	if CheckGitVersionAtLeast("2.18") == nil {
		cmd := NewCommand(ctx, "commit-graph", "write")
		if CheckGitVersionAtLeast("2.27") == nil {
			cmd.AddArguments("--changed-paths")
		}
		if _, _, err := cmd.RunStdString(&RunOpts{Dir: repoPath}); err != nil {
			return fmt.Errorf("unable to write commit-graph for '%s' : %w", repoPath, err)
		}
	}
	return nil
}
