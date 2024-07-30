// Copyright 2024 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package git

import (
	"bytes"
	"context"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGrepSearch(t *testing.T) {
	repo, err := openRepositoryWithDefaultContext(filepath.Join(testReposDir, "language_stats_repo"))
	require.NoError(t, err)
	defer repo.Close()

	res, err := GrepSearch(context.Background(), repo, "void", GrepOptions{})
	require.NoError(t, err)
	assert.Equal(t, []*GrepResult{
		{
			Filename:    "java-hello/main.java",
			LineNumbers: []int{3},
			LineCodes:   []string{" public static void main(String[] args)"},
		},
		{
			Filename:    "main.vendor.java",
			LineNumbers: []int{3},
			LineCodes:   []string{" public static void main(String[] args)"},
		},
	}, res)

	res, err = GrepSearch(context.Background(), repo, "void", GrepOptions{MaxResultLimit: 1})
	require.NoError(t, err)
	assert.Equal(t, []*GrepResult{
		{
			Filename:    "java-hello/main.java",
			LineNumbers: []int{3},
			LineCodes:   []string{" public static void main(String[] args)"},
		},
	}, res)

	res, err = GrepSearch(context.Background(), repo, "world", GrepOptions{MatchesPerFile: 1})
	require.NoError(t, err)
	assert.Equal(t, []*GrepResult{
		{
			Filename:    "i-am-a-python.p",
			LineNumbers: []int{1},
			LineCodes:   []string{"## This is a simple file to do a hello world"},
		},
		{
			Filename:    "java-hello/main.java",
			LineNumbers: []int{1},
			LineCodes:   []string{"public class HelloWorld"},
		},
		{
			Filename:    "main.vendor.java",
			LineNumbers: []int{1},
			LineCodes:   []string{"public class HelloWorld"},
		},
		{
			Filename:    "python-hello/hello.py",
			LineNumbers: []int{1},
			LineCodes:   []string{"## This is a simple file to do a hello world"},
		},
	}, res)

	res, err = GrepSearch(context.Background(), repo, "no-such-content", GrepOptions{})
	require.NoError(t, err)
	assert.Empty(t, res)

	res, err = GrepSearch(context.Background(), &Repository{Path: "no-such-git-repo"}, "no-such-content", GrepOptions{})
	require.Error(t, err)
	assert.Empty(t, res)
}

func TestGrepLongFiles(t *testing.T) {
	tmpDir := t.TempDir()

	err := InitRepository(DefaultContext, tmpDir, false, Sha1ObjectFormat.Name())
	require.NoError(t, err)

	gitRepo, err := openRepositoryWithDefaultContext(tmpDir)
	require.NoError(t, err)
	defer gitRepo.Close()

	require.NoError(t, os.WriteFile(path.Join(tmpDir, "README.md"), bytes.Repeat([]byte{'a'}, 65*1024), 0o666))

	err = AddChanges(tmpDir, true)
	require.NoError(t, err)

	err = CommitChanges(tmpDir, CommitChangesOptions{Message: "Long file"})
	require.NoError(t, err)

	res, err := GrepSearch(context.Background(), gitRepo, "a", GrepOptions{})
	require.NoError(t, err)
	assert.Len(t, res, 1)
	assert.Len(t, res[0].LineCodes[0], 65*1024)
}

func TestGrepRefs(t *testing.T) {
	tmpDir := t.TempDir()

	err := InitRepository(DefaultContext, tmpDir, false, Sha1ObjectFormat.Name())
	require.NoError(t, err)

	gitRepo, err := openRepositoryWithDefaultContext(tmpDir)
	require.NoError(t, err)
	defer gitRepo.Close()

	require.NoError(t, os.WriteFile(path.Join(tmpDir, "README.md"), []byte{'A'}, 0o666))
	require.NoError(t, AddChanges(tmpDir, true))

	err = CommitChanges(tmpDir, CommitChangesOptions{Message: "add A"})
	require.NoError(t, err)

	require.NoError(t, gitRepo.CreateTag("v1", "HEAD"))

	require.NoError(t, os.WriteFile(path.Join(tmpDir, "README.md"), []byte{'A', 'B', 'C', 'D'}, 0o666))
	require.NoError(t, AddChanges(tmpDir, true))

	err = CommitChanges(tmpDir, CommitChangesOptions{Message: "add BCD"})
	require.NoError(t, err)

	res, err := GrepSearch(context.Background(), gitRepo, "a", GrepOptions{RefName: "v1"})
	require.NoError(t, err)
	assert.Len(t, res, 1)
	assert.Equal(t, "A", res[0].LineCodes[0])
}
