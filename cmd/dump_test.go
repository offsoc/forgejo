// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package cmd

import (
	"io"
	"os"
	"testing"

	"github.com/mholt/archiver/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockArchiver struct {
	addedFiles []string
}

func (mockArchiver) Create(out io.Writer) error {
	return nil
}

func (m *mockArchiver) Write(f archiver.File) error {
	m.addedFiles = append(m.addedFiles, f.Name())
	return nil
}

func (mockArchiver) Close() error {
	return nil
}

func TestAddRecursiveExclude(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		dir := t.TempDir()
		archiver := &mockArchiver{}

		err := addRecursiveExclude(archiver, "", dir, []string{}, false)
		require.NoError(t, err)
		assert.Empty(t, archiver.addedFiles)
	})

	t.Run("Single file", func(t *testing.T) {
		dir := t.TempDir()
		err := os.WriteFile(dir+"/example", nil, 0o666)
		require.NoError(t, err)

		t.Run("No exclude", func(t *testing.T) {
			archiver := &mockArchiver{}

			err = addRecursiveExclude(archiver, "", dir, nil, false)
			require.NoError(t, err)
			assert.Len(t, archiver.addedFiles, 1)
			assert.Contains(t, archiver.addedFiles, "example")
		})

		t.Run("With exclude", func(t *testing.T) {
			archiver := &mockArchiver{}

			err = addRecursiveExclude(archiver, "", dir, []string{dir + "/example"}, false)
			require.NoError(t, err)
			assert.Empty(t, archiver.addedFiles)
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
			archiver := &mockArchiver{}

			err = addRecursiveExclude(archiver, "", dir, nil, false)
			require.NoError(t, err)
			assert.Len(t, archiver.addedFiles, 5)
			assert.Contains(t, archiver.addedFiles, "deep")
			assert.Contains(t, archiver.addedFiles, "deep/nested")
			assert.Contains(t, archiver.addedFiles, "deep/nested/folder")
			assert.Contains(t, archiver.addedFiles, "deep/nested/folder/example")
			assert.Contains(t, archiver.addedFiles, "deep/nested/folder/another-file")
		})

		t.Run("Exclude first directory", func(t *testing.T) {
			archiver := &mockArchiver{}

			err = addRecursiveExclude(archiver, "", dir, []string{dir + "/deep"}, false)
			require.NoError(t, err)
			assert.Empty(t, archiver.addedFiles)
		})

		t.Run("Exclude nested directory", func(t *testing.T) {
			archiver := &mockArchiver{}

			err = addRecursiveExclude(archiver, "", dir, []string{dir + "/deep/nested/folder"}, false)
			require.NoError(t, err)
			assert.Len(t, archiver.addedFiles, 2)
			assert.Contains(t, archiver.addedFiles, "deep")
			assert.Contains(t, archiver.addedFiles, "deep/nested")
		})

		t.Run("Exclude file", func(t *testing.T) {
			archiver := &mockArchiver{}

			err = addRecursiveExclude(archiver, "", dir, []string{dir + "/deep/nested/folder/example"}, false)
			require.NoError(t, err)
			assert.Len(t, archiver.addedFiles, 4)
			assert.Contains(t, archiver.addedFiles, "deep")
			assert.Contains(t, archiver.addedFiles, "deep/nested")
			assert.Contains(t, archiver.addedFiles, "deep/nested/folder")
			assert.Contains(t, archiver.addedFiles, "deep/nested/folder/another-file")
		})
	})
}
