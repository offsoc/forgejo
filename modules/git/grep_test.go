// Copyright 2024 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package git

import (
	"bytes"
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

	res, err := GrepSearch(t.Context(), repo, "public", GrepOptions{})
	require.NoError(t, err)
	assert.Equal(t, []*GrepResult{
		{
			Filename:    "java-hello/main.java",
			LineNumbers: []int{1, 3},
			LineCodes: []string{
				"public class HelloWorld",
				" public static void main(String[] args)",
			},
			HighlightedRanges: [][3]int{{0, 0, 6}, {1, 1, 7}},
		},
		{
			Filename:    "main.vendor.java",
			LineNumbers: []int{1, 3},
			LineCodes: []string{
				"public class HelloWorld",
				" public static void main(String[] args)",
			},
			HighlightedRanges: [][3]int{{0, 0, 6}, {1, 1, 7}},
		},
	}, res)

	res, err = GrepSearch(t.Context(), repo, "void", GrepOptions{MaxResultLimit: 1, ContextLineNumber: 2})
	require.NoError(t, err)
	assert.Equal(t, []*GrepResult{
		{
			Filename:    "java-hello/main.java",
			LineNumbers: []int{1, 2, 3, 4, 5},
			LineCodes: []string{
				"public class HelloWorld",
				"{",
				" public static void main(String[] args)",
				" {",
				"  System.out.println(\"Hello world!\");",
			},
			HighlightedRanges: [][3]int{{2, 15, 19}},
		},
	}, res)

	res, err = GrepSearch(t.Context(), repo, "world", GrepOptions{MatchesPerFile: 1})
	require.NoError(t, err)
	assert.Equal(t, []*GrepResult{
		{
			Filename:          "i-am-a-python.p",
			LineNumbers:       []int{1},
			LineCodes:         []string{"## This is a simple file to do a hello world"},
			HighlightedRanges: [][3]int{{0, 39, 44}},
		},
		{
			Filename:          "java-hello/main.java",
			LineNumbers:       []int{1},
			LineCodes:         []string{"public class HelloWorld"},
			HighlightedRanges: [][3]int{{0, 18, 23}},
		},
		{
			Filename:          "main.vendor.java",
			LineNumbers:       []int{1},
			LineCodes:         []string{"public class HelloWorld"},
			HighlightedRanges: [][3]int{{0, 18, 23}},
		},
		{
			Filename:          "python-hello/hello.py",
			LineNumbers:       []int{1},
			LineCodes:         []string{"## This is a simple file to do a hello world"},
			HighlightedRanges: [][3]int{{0, 39, 44}},
		},
	}, res)

	res, err = GrepSearch(t.Context(), repo, "world", GrepOptions{
		MatchesPerFile: 1,
		Filename:       "java-hello/",
	})
	require.NoError(t, err)
	assert.Equal(t, []*GrepResult{
		{
			Filename:          "java-hello/main.java",
			LineNumbers:       []int{1},
			LineCodes:         []string{"public class HelloWorld"},
			HighlightedRanges: [][3]int{{0, 18, 23}},
		},
	}, res)

	res, err = GrepSearch(t.Context(), repo, "no-such-content", GrepOptions{})
	require.NoError(t, err)
	assert.Empty(t, res)

	res, err = GrepSearch(t.Context(), &Repository{Path: "no-such-git-repo"}, "no-such-content", GrepOptions{})
	require.Error(t, err)
	assert.Empty(t, res)
}

func TestGrepDashesAreFine(t *testing.T) {
	tmpDir := t.TempDir()

	err := InitRepository(DefaultContext, tmpDir, false, Sha1ObjectFormat.Name())
	require.NoError(t, err)

	gitRepo, err := openRepositoryWithDefaultContext(tmpDir)
	require.NoError(t, err)
	defer gitRepo.Close()

	require.NoError(t, os.WriteFile(path.Join(tmpDir, "with-dashes"), []byte("--"), 0o666))
	require.NoError(t, os.WriteFile(path.Join(tmpDir, "without-dashes"), []byte(".."), 0o666))

	err = AddChanges(tmpDir, true)
	require.NoError(t, err)

	err = CommitChanges(tmpDir, CommitChangesOptions{Message: "Dashes are cool sometimes"})
	require.NoError(t, err)

	res, err := GrepSearch(t.Context(), gitRepo, "--", GrepOptions{})
	require.NoError(t, err)
	assert.Len(t, res, 1)
	assert.Equal(t, "with-dashes", res[0].Filename)
}

func TestGrepNoBinary(t *testing.T) {
	tmpDir := t.TempDir()

	err := InitRepository(DefaultContext, tmpDir, false, Sha1ObjectFormat.Name())
	require.NoError(t, err)

	gitRepo, err := openRepositoryWithDefaultContext(tmpDir)
	require.NoError(t, err)
	defer gitRepo.Close()

	require.NoError(t, os.WriteFile(path.Join(tmpDir, "BINARY"), []byte("I AM BINARY\n\x00\nYOU WON'T SEE ME"), 0o666))
	require.NoError(t, os.WriteFile(path.Join(tmpDir, "TEXT"), []byte("I AM NOT BINARY\nYOU WILL SEE ME"), 0o666))

	err = AddChanges(tmpDir, true)
	require.NoError(t, err)

	err = CommitChanges(tmpDir, CommitChangesOptions{Message: "Binary and text files"})
	require.NoError(t, err)

	res, err := GrepSearch(t.Context(), gitRepo, "BINARY", GrepOptions{})
	require.NoError(t, err)
	assert.Len(t, res, 1)
	assert.Equal(t, "TEXT", res[0].Filename)
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

	res, err := GrepSearch(t.Context(), gitRepo, "a", GrepOptions{})
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

	res, err := GrepSearch(t.Context(), gitRepo, "a", GrepOptions{RefName: "v1"})
	require.NoError(t, err)
	assert.Len(t, res, 1)
	assert.Equal(t, "A", res[0].LineCodes[0])
}

func TestGrepCanHazRegexOnDemand(t *testing.T) {
	tmpDir := t.TempDir()

	err := InitRepository(DefaultContext, tmpDir, false, Sha1ObjectFormat.Name())
	require.NoError(t, err)

	gitRepo, err := openRepositoryWithDefaultContext(tmpDir)
	require.NoError(t, err)
	defer gitRepo.Close()

	require.NoError(t, os.WriteFile(path.Join(tmpDir, "matching"), []byte("It's a match!"), 0o666))
	require.NoError(t, os.WriteFile(path.Join(tmpDir, "not-matching"), []byte("Orisitamatch?"), 0o666))

	err = AddChanges(tmpDir, true)
	require.NoError(t, err)

	err = CommitChanges(tmpDir, CommitChangesOptions{Message: "Add fixtures for regexp test"})
	require.NoError(t, err)

	// should find nothing by default...
	res, err := GrepSearch(t.Context(), gitRepo, "\\bmatch\\b", GrepOptions{})
	require.NoError(t, err)
	assert.Empty(t, res)

	// ... unless configured explicitly
	res, err = GrepSearch(t.Context(), gitRepo, "\\bmatch\\b", GrepOptions{Mode: RegExpGrepMode})
	require.NoError(t, err)
	assert.Len(t, res, 1)
	assert.Equal(t, "matching", res[0].Filename)
}
