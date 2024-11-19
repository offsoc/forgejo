// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package lfs

import (
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStringContent(t *testing.T) {
	p := Pointer{Oid: "4d7a214614ab2935c943f9e0ff69d22eadbb8f32b1258daaa5e2ca24d17e2393", Size: 1234}
	expected := "version https://git-lfs.github.com/spec/v1\noid sha256:4d7a214614ab2935c943f9e0ff69d22eadbb8f32b1258daaa5e2ca24d17e2393\nsize 1234\n"
	assert.Equal(t, expected, p.StringContent())
}

func TestRelativePath(t *testing.T) {
	p := Pointer{Oid: "4d7a214614ab2935c943f9e0ff69d22eadbb8f32b1258daaa5e2ca24d17e2393"}
	expected := path.Join("4d", "7a", "214614ab2935c943f9e0ff69d22eadbb8f32b1258daaa5e2ca24d17e2393")
	assert.Equal(t, expected, p.RelativePath())

	p2 := Pointer{Oid: "4d7a"}
	assert.Equal(t, "4d7a", p2.RelativePath())
}

func TestIsValid(t *testing.T) {
	p := Pointer{}
	assert.False(t, p.IsValid())

	p = Pointer{Oid: "123"}
	assert.False(t, p.IsValid())

	p = Pointer{Oid: "z4cb57646c54a297c9807697e80a30946f79a4b82cb079d2606847825b1812cc"}
	assert.False(t, p.IsValid())

	p = Pointer{Oid: "94cb57646c54a297c9807697e80a30946f79a4b82cb079d2606847825b1812cc"}
	assert.True(t, p.IsValid())

	p = Pointer{Oid: "94cb57646c54a297c9807697e80a30946f79a4b82cb079d2606847825b1812cc", Size: -1}
	assert.False(t, p.IsValid())
}

func TestGeneratePointer(t *testing.T) {
	p, err := GeneratePointer(strings.NewReader("Gitea"))
	require.NoError(t, err)
	assert.True(t, p.IsValid())
	assert.Equal(t, "94cb57646c54a297c9807697e80a30946f79a4b82cb079d2606847825b1812cc", p.Oid)
	assert.Equal(t, int64(5), p.Size)
}

func TestReadPointerFromBuffer(t *testing.T) {
	p, err := ReadPointerFromBuffer([]byte{})
	require.ErrorIs(t, err, ErrMissingPrefix)
	assert.False(t, p.IsValid())

	p, err = ReadPointerFromBuffer([]byte("test"))
	require.ErrorIs(t, err, ErrMissingPrefix)
	assert.False(t, p.IsValid())

	p, err = ReadPointerFromBuffer([]byte("version https://git-lfs.github.com/spec/v1\n"))
	require.ErrorIs(t, err, ErrInvalidStructure)
	assert.False(t, p.IsValid())

	p, err = ReadPointerFromBuffer([]byte("version https://git-lfs.github.com/spec/v1\noid sha256:4d7a\nsize 1234\n"))
	require.ErrorIs(t, err, ErrInvalidOIDFormat)
	assert.False(t, p.IsValid())

	p, err = ReadPointerFromBuffer([]byte("version https://git-lfs.github.com/spec/v1\noid sha256:4d7a2146z4ab2935c943f9e0ff69d22eadbb8f32b1258daaa5e2ca24d17e2393\nsize 1234\n"))
	require.ErrorIs(t, err, ErrInvalidOIDFormat)
	assert.False(t, p.IsValid())

	p, err = ReadPointerFromBuffer([]byte("version https://git-lfs.github.com/spec/v1\noid sha256:4d7a214614ab2935c943f9e0ff69d22eadbb8f32b1258daaa5e2ca24d17e2393\ntest 1234\n"))
	require.Error(t, err)
	assert.False(t, p.IsValid())

	p, err = ReadPointerFromBuffer([]byte("version https://git-lfs.github.com/spec/v1\noid sha256:4d7a214614ab2935c943f9e0ff69d22eadbb8f32b1258daaa5e2ca24d17e2393\nsize test\n"))
	require.Error(t, err)
	assert.False(t, p.IsValid())

	p, err = ReadPointerFromBuffer([]byte("version https://git-lfs.github.com/spec/v1\noid sha256:4d7a214614ab2935c943f9e0ff69d22eadbb8f32b1258daaa5e2ca24d17e2393\nsize 1234\n"))
	require.NoError(t, err)
	assert.True(t, p.IsValid())
	assert.Equal(t, "4d7a214614ab2935c943f9e0ff69d22eadbb8f32b1258daaa5e2ca24d17e2393", p.Oid)
	assert.Equal(t, int64(1234), p.Size)

	p, err = ReadPointerFromBuffer([]byte("version https://git-lfs.github.com/spec/v1\noid sha256:4d7a214614ab2935c943f9e0ff69d22eadbb8f32b1258daaa5e2ca24d17e2393\nsize 1234\ntest"))
	require.NoError(t, err)
	assert.True(t, p.IsValid())
	assert.Equal(t, "4d7a214614ab2935c943f9e0ff69d22eadbb8f32b1258daaa5e2ca24d17e2393", p.Oid)
	assert.Equal(t, int64(1234), p.Size)
}

func TestReadPointer(t *testing.T) {
	p, err := ReadPointer(strings.NewReader("version https://git-lfs.github.com/spec/v1\noid sha256:4d7a214614ab2935c943f9e0ff69d22eadbb8f32b1258daaa5e2ca24d17e2393\nsize 1234\n"))
	require.NoError(t, err)
	assert.True(t, p.IsValid())
	assert.Equal(t, "4d7a214614ab2935c943f9e0ff69d22eadbb8f32b1258daaa5e2ca24d17e2393", p.Oid)
	assert.Equal(t, int64(1234), p.Size)
}
