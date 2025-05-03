// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package base

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodeSha256(t *testing.T) {
	assert.Equal(t,
		"c3ab8ff13720e8ad9047dd39466b3c8974e592c2fa383d4a3960714caef0c4f2",
		EncodeSha256("foobar"),
	)
}

func TestShortSha(t *testing.T) {
	assert.Equal(t, "veryverylo", ShortSha("veryverylong"))
}

func TestBasicAuthDecode(t *testing.T) {
	_, _, err := BasicAuthDecode("?")
	assert.Equal(t, "illegal base64 data at input byte 0", err.Error())

	user, pass, err := BasicAuthDecode("Zm9vOmJhcg==")
	require.NoError(t, err)
	assert.Equal(t, "foo", user)
	assert.Equal(t, "bar", pass)

	_, _, err = BasicAuthDecode("aW52YWxpZA==")
	require.Error(t, err)

	_, _, err = BasicAuthDecode("invalid")
	require.Error(t, err)

	_, _, err = BasicAuthDecode("YWxpY2U=") // "alice", no colon
	require.Error(t, err)
}

func TestFileSize(t *testing.T) {
	var size int64 = 512
	assert.Equal(t, "512 B", FileSize(size))
	size *= 1024
	assert.Equal(t, "512 KiB", FileSize(size))
	size *= 1024
	assert.Equal(t, "512 MiB", FileSize(size))
	size *= 1024
	assert.Equal(t, "512 GiB", FileSize(size))
	size *= 1024
	assert.Equal(t, "512 TiB", FileSize(size))
	size *= 1024
	assert.Equal(t, "512 PiB", FileSize(size))
	size *= 4
	assert.Equal(t, "2.0 EiB", FileSize(size))
}

func TestEllipsisString(t *testing.T) {
	assert.Equal(t, "...", EllipsisString("foobar", 0))
	assert.Equal(t, "...", EllipsisString("foobar", 1))
	assert.Equal(t, "...", EllipsisString("foobar", 2))
	assert.Equal(t, "...", EllipsisString("foobar", 3))
	assert.Equal(t, "f...", EllipsisString("foobar", 4))
	assert.Equal(t, "fo...", EllipsisString("foobar", 5))
	assert.Equal(t, "foobar", EllipsisString("foobar", 6))
	assert.Equal(t, "foobar", EllipsisString("foobar", 10))
	assert.Equal(t, "测...", EllipsisString("测试文本一二三四", 4))
	assert.Equal(t, "测试...", EllipsisString("测试文本一二三四", 5))
	assert.Equal(t, "测试文...", EllipsisString("测试文本一二三四", 6))
	assert.Equal(t, "测试文本一二三四", EllipsisString("测试文本一二三四", 10))
}

func TestTruncateString(t *testing.T) {
	assert.Empty(t, TruncateString("foobar", 0))
	assert.Equal(t, "f", TruncateString("foobar", 1))
	assert.Equal(t, "fo", TruncateString("foobar", 2))
	assert.Equal(t, "foo", TruncateString("foobar", 3))
	assert.Equal(t, "foob", TruncateString("foobar", 4))
	assert.Equal(t, "fooba", TruncateString("foobar", 5))
	assert.Equal(t, "foobar", TruncateString("foobar", 6))
	assert.Equal(t, "foobar", TruncateString("foobar", 7))
	assert.Equal(t, "测试文本", TruncateString("测试文本一二三四", 4))
	assert.Equal(t, "测试文本一", TruncateString("测试文本一二三四", 5))
	assert.Equal(t, "测试文本一二", TruncateString("测试文本一二三四", 6))
	assert.Equal(t, "测试文本一二三", TruncateString("测试文本一二三四", 7))
}

func TestStringsToInt64s(t *testing.T) {
	testSuccess := func(input []string, expected []int64) {
		result, err := StringsToInt64s(input)
		require.NoError(t, err)
		assert.Equal(t, expected, result)
	}
	testSuccess(nil, nil)
	testSuccess([]string{}, []int64{})
	testSuccess([]string{"-1234"}, []int64{-1234})
	testSuccess([]string{"1", "4", "16", "64", "256"}, []int64{1, 4, 16, 64, 256})

	ints, err := StringsToInt64s([]string{"-1", "a"})
	assert.Empty(t, ints)
	require.Error(t, err)
}

func TestInt64sToStrings(t *testing.T) {
	assert.Equal(t, []string{}, Int64sToStrings([]int64{}))
	assert.Equal(t,
		[]string{"1", "4", "16", "64", "256"},
		Int64sToStrings([]int64{1, 4, 16, 64, 256}),
	)
}

// TODO: Test EntryIcon

func TestSetupGiteaRoot(t *testing.T) {
	t.Setenv("GITEA_ROOT", "test")
	assert.Equal(t, "test", SetupGiteaRoot())
	t.Setenv("GITEA_ROOT", "")
	assert.NotEqual(t, "test", SetupGiteaRoot())
}

func TestFormatNumberSI(t *testing.T) {
	assert.Equal(t, "125", FormatNumberSI(int(125)))
	assert.Equal(t, "1.3k", FormatNumberSI(int64(1317)))
	assert.Equal(t, "21.3M", FormatNumberSI(21317675))
	assert.Equal(t, "45.7G", FormatNumberSI(45721317675))
	assert.Empty(t, FormatNumberSI("test"))
}
