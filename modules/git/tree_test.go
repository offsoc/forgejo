// Copyright 2024 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package git

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubTree_Issue29101(t *testing.T) {
	repo, err := openRepositoryWithDefaultContext(filepath.Join(testReposDir, "repo1_bare"))
	require.NoError(t, err)
	defer repo.Close()

	commit, err := repo.GetCommit("ce064814f4a0d337b333e646ece456cd39fab612")
	require.NoError(t, err)

	// old code could produce a different error if called multiple times
	for i := 0; i < 10; i++ {
		_, err = commit.SubTree("file1.txt")
		require.Error(t, err)
		assert.True(t, IsErrNotExist(err))
	}
}
