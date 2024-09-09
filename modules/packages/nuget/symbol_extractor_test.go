// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package nuget

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const pdbContent = `QlNKQgEAAQAAAAAADAAAAFBEQiB2MS4wAAAAAAAABgB8AAAAWAAAACNQZGIAAAAA1AAAAAgBAAAj
fgAA3AEAAAQAAAAjU3RyaW5ncwAAAADgAQAABAAAACNVUwDkAQAAMAAAACNHVUlEAAAAFAIAACgB
AAAjQmxvYgAAAGm7ENm9SGxMtAFVvPUsPJTF6PbtAAAAAFcVogEJAAAAAQAAAA==`

func TestExtractPortablePdb(t *testing.T) {
	createArchive := func(name string, content []byte) []byte {
		var buf bytes.Buffer
		archive := zip.NewWriter(&buf)
		w, _ := archive.Create(name)
		w.Write(content)
		archive.Close()
		return buf.Bytes()
	}

	t.Run("MissingPdbFiles", func(t *testing.T) {
		var buf bytes.Buffer
		zip.NewWriter(&buf).Close()

		pdbs, err := ExtractPortablePdb(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
		require.ErrorIs(t, err, ErrMissingPdbFiles)
		assert.Empty(t, pdbs)
	})

	t.Run("InvalidFiles", func(t *testing.T) {
		data := createArchive("sub/test.bin", []byte{})

		pdbs, err := ExtractPortablePdb(bytes.NewReader(data), int64(len(data)))
		require.ErrorIs(t, err, ErrInvalidFiles)
		assert.Empty(t, pdbs)
	})

	t.Run("Valid", func(t *testing.T) {
		b, _ := base64.StdEncoding.DecodeString(pdbContent)
		data := createArchive("test.pdb", b)

		pdbs, err := ExtractPortablePdb(bytes.NewReader(data), int64(len(data)))
		require.NoError(t, err)
		assert.Len(t, pdbs, 1)
		assert.Equal(t, "test.pdb", pdbs[0].Name)
		assert.Equal(t, "d910bb6948bd4c6cb40155bcf52c3c94", pdbs[0].ID)
		pdbs.Close()
	})
}

func TestParseDebugHeaderID(t *testing.T) {
	t.Run("InvalidPdbMagicNumber", func(t *testing.T) {
		id, err := ParseDebugHeaderID(bytes.NewReader([]byte{0, 0, 0, 0}))
		require.ErrorIs(t, err, ErrInvalidPdbMagicNumber)
		assert.Empty(t, id)
	})

	t.Run("MissingPdbStream", func(t *testing.T) {
		b, _ := base64.StdEncoding.DecodeString(`QlNKQgEAAQAAAAAADAAAAFBEQiB2MS4wAAAAAAAAAQB8AAAAWAAAACNVUwA=`)

		id, err := ParseDebugHeaderID(bytes.NewReader(b))
		require.ErrorIs(t, err, ErrMissingPdbStream)
		assert.Empty(t, id)
	})

	t.Run("Valid", func(t *testing.T) {
		b, _ := base64.StdEncoding.DecodeString(pdbContent)

		id, err := ParseDebugHeaderID(bytes.NewReader(b))
		require.NoError(t, err)
		assert.Equal(t, "d910bb6948bd4c6cb40155bcf52c3c94", id)
	})
}
