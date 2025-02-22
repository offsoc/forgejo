// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package cmd

import (
	"os"
	"testing"

	"github.com/mholt/archives"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mockArchiverAsync(ch chan archives.ArchiveAsyncJob, files *[]string) {
	for job := range ch {
		*files = append(*files, job.File.NameInArchive)
		job.Result <- nil
	}
}

func TestAddRecursiveExclude(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		ch := make(chan archives.ArchiveAsyncJob)
		var files []string
		go mockArchiverAsync(ch, &files)

		dir := t.TempDir()

		err := addRecursiveExclude(ch, "", dir, []string{}, false)
		require.NoError(t, err)
		assert.Empty(t, files)
	})

	t.Run("Single file", func(t *testing.T) {
		dir := t.TempDir()
		err := os.WriteFile(dir+"/example", nil, 0o666)
		require.NoError(t, err)

		t.Run("No exclude", func(t *testing.T) {
			ch := make(chan archives.ArchiveAsyncJob)
			var files []string
			go mockArchiverAsync(ch, &files)

			err := addRecursiveExclude(ch, "", dir, nil, false)
			require.NoError(t, err)

			assert.Len(t, files, 1)
			assert.Contains(t, files, "example")
		})

		t.Run("With exclude", func(t *testing.T) {
			ch := make(chan archives.ArchiveAsyncJob)
			var files []string
			go mockArchiverAsync(ch, &files)

			err := addRecursiveExclude(ch, "", dir, []string{dir + "/example"}, false)
			require.NoError(t, err)
			assert.Empty(t, files)
		})
	})

	t.Run("File inside directory", func(t *testing.T) {
		dir := t.TempDir()
		err := os.MkdirAll(dir+"/deep/nested/folder", 0o750)
		require.NoError(t, err)
		err = os.WriteFile(dir+"/deep/nested/folder/example", nil, 0o666)
		require.NoError(t, err)
		err = os.WriteFile(dir+"/deep/nested/folder/another-file", nil, 0o666)
		require.NoError(t, err)

		t.Run("No exclude", func(t *testing.T) {
			ch := make(chan archives.ArchiveAsyncJob)
			var files []string
			go mockArchiverAsync(ch, &files)

			err := addRecursiveExclude(ch, "", dir, nil, false)
			require.NoError(t, err)
			assert.Len(t, files, 5)

			assert.Contains(t, files, "deep")
			assert.Contains(t, files, "deep/nested")
			assert.Contains(t, files, "deep/nested/folder")
			assert.Contains(t, files, "deep/nested/folder/example")
			assert.Contains(t, files, "deep/nested/folder/another-file")
		})

		t.Run("Exclude first directory", func(t *testing.T) {
			ch := make(chan archives.ArchiveAsyncJob)
			var files []string
			go mockArchiverAsync(ch, &files)

			err := addRecursiveExclude(ch, "", dir, []string{dir + "/deep"}, false)
			require.NoError(t, err)
			assert.Empty(t, files)
		})

		t.Run("Exclude nested directory", func(t *testing.T) {
			ch := make(chan archives.ArchiveAsyncJob)
			var files []string
			go mockArchiverAsync(ch, &files)

			err := addRecursiveExclude(ch, "", dir, []string{dir + "/deep/nested/folder"}, false)
			require.NoError(t, err)
			assert.Len(t, files, 2)

			assert.Contains(t, files, "deep")
			assert.Contains(t, files, "deep/nested")
		})

		t.Run("Exclude file", func(t *testing.T) {
			ch := make(chan archives.ArchiveAsyncJob)
			var files []string
			go mockArchiverAsync(ch, &files)

			err := addRecursiveExclude(ch, "", dir, []string{dir + "/deep/nested/folder/example"}, false)
			require.NoError(t, err)
			assert.Len(t, files, 4)

			assert.Contains(t, files, "deep")
			assert.Contains(t, files, "deep/nested")
			assert.Contains(t, files, "deep/nested/folder")
			assert.Contains(t, files, "deep/nested/folder/another-file")
		})
	})
}
