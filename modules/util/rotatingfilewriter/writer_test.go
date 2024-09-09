// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package rotatingfilewriter

import (
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompressOldFile(t *testing.T) {
	tmpDir := t.TempDir()
	fname := filepath.Join(tmpDir, "test")
	nonGzip := filepath.Join(tmpDir, "test-nonGzip")

	f, err := os.OpenFile(fname, os.O_CREATE|os.O_WRONLY, 0o660)
	require.NoError(t, err)
	ng, err := os.OpenFile(nonGzip, os.O_CREATE|os.O_WRONLY, 0o660)
	require.NoError(t, err)

	for i := 0; i < 999; i++ {
		f.WriteString("This is a test file\n")
		ng.WriteString("This is a test file\n")
	}
	f.Close()
	ng.Close()

	err = compressOldFile(fname, gzip.DefaultCompression)
	require.NoError(t, err)

	_, err = os.Lstat(fname + ".gz")
	require.NoError(t, err)

	f, err = os.Open(fname + ".gz")
	require.NoError(t, err)
	zr, err := gzip.NewReader(f)
	require.NoError(t, err)
	data, err := io.ReadAll(zr)
	require.NoError(t, err)
	original, err := os.ReadFile(nonGzip)
	require.NoError(t, err)
	assert.Equal(t, original, data)
}
