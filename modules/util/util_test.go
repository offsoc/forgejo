// Copyright 2018 The Gitea Authors. All rights reserved.
// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package util_test

import (
	"bytes"
	"crypto/rand"
	"regexp"
	"strings"
	"testing"

	"forgejo.org/modules/test"
	"forgejo.org/modules/util"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestURLJoin(t *testing.T) {
	type test struct {
		Expected string
		Base     string
		Elements []string
	}
	newTest := func(expected, base string, elements ...string) test {
		return test{Expected: expected, Base: base, Elements: elements}
	}
	for _, test := range []test{
		newTest("https://try.gitea.io/a/b/c",
			"https://try.gitea.io", "a/b", "c"),
		newTest("https://try.gitea.io/a/b/c",
			"https://try.gitea.io/", "/a/b/", "/c/"),
		newTest("https://try.gitea.io/a/c",
			"https://try.gitea.io/", "/a/./b/", "../c/"),
		newTest("a/b/c",
			"a", "b/c/"),
		newTest("a/b/d",
			"a/", "b/c/", "/../d/"),
		newTest("https://try.gitea.io/a/b/c#d",
			"https://try.gitea.io", "a/b", "c#d"),
		newTest("/a/b/d",
			"/a/", "b/c/", "/../d/"),
		newTest("/a/b/c",
			"/a", "b/c/"),
		newTest("/a/b/c#hash",
			"/a", "b/c#hash"),
	} {
		assert.Equal(t, test.Expected, util.URLJoin(test.Base, test.Elements...))
	}
}

func TestIsEmptyString(t *testing.T) {
	cases := []struct {
		s        string
		expected bool
	}{
		{"", true},
		{" ", true},
		{"   ", true},
		{"  a", false},
	}

	for _, v := range cases {
		assert.Equal(t, v.expected, util.IsEmptyString(v.s))
	}
}

func Test_NormalizeEOL(t *testing.T) {
	data1 := []string{
		"",
		"This text starts with empty lines",
		"another",
		"",
		"",
		"",
		"Some other empty lines in the middle",
		"more.",
		"And more.",
		"Ends with empty lines too.",
		"",
		"",
		"",
	}

	data2 := []string{
		"This text does not start with empty lines",
		"another",
		"",
		"",
		"",
		"Some other empty lines in the middle",
		"more.",
		"And more.",
		"Ends without EOLtoo.",
	}

	buildEOLData := func(data []string, eol string) []byte {
		return []byte(strings.Join(data, eol))
	}

	dos := buildEOLData(data1, "\r\n")
	unix := buildEOLData(data1, "\n")
	mac := buildEOLData(data1, "\r")

	assert.Equal(t, unix, util.NormalizeEOL(dos))
	assert.Equal(t, unix, util.NormalizeEOL(mac))
	assert.Equal(t, unix, util.NormalizeEOL(unix))

	dos = buildEOLData(data2, "\r\n")
	unix = buildEOLData(data2, "\n")
	mac = buildEOLData(data2, "\r")

	assert.Equal(t, unix, util.NormalizeEOL(dos))
	assert.Equal(t, unix, util.NormalizeEOL(mac))
	assert.Equal(t, unix, util.NormalizeEOL(unix))

	assert.Equal(t, []byte("one liner"), util.NormalizeEOL([]byte("one liner")))
	assert.Equal(t, []byte("\n"), util.NormalizeEOL([]byte("\n")))
	assert.Equal(t, []byte("\ntwo liner"), util.NormalizeEOL([]byte("\ntwo liner")))
	assert.Equal(t, []byte("two liner\n"), util.NormalizeEOL([]byte("two liner\n")))
	assert.Equal(t, []byte{}, util.NormalizeEOL([]byte{}))

	assert.Equal(t, []byte("mix\nand\nmatch\n."), util.NormalizeEOL([]byte("mix\r\nand\rmatch\n.")))
}

func Test_RandomInt(t *testing.T) {
	randInt, err := util.CryptoRandomInt(255)
	assert.GreaterOrEqual(t, randInt, int64(0))
	assert.LessOrEqual(t, randInt, int64(255))
	require.NoError(t, err)
}

func Test_RandomString(t *testing.T) {
	str1, err := util.CryptoRandomString(32)
	require.NoError(t, err)
	matches, err := regexp.MatchString(`^[a-zA-Z0-9]{32}$`, str1)
	require.NoError(t, err)
	assert.True(t, matches)

	str2, err := util.CryptoRandomString(32)
	require.NoError(t, err)
	matches, err = regexp.MatchString(`^[a-zA-Z0-9]{32}$`, str1)
	require.NoError(t, err)
	assert.True(t, matches)

	assert.NotEqual(t, str1, str2)

	str3, err := util.CryptoRandomString(256)
	require.NoError(t, err)
	matches, err = regexp.MatchString(`^[a-zA-Z0-9]{256}$`, str3)
	require.NoError(t, err)
	assert.True(t, matches)

	str4, err := util.CryptoRandomString(256)
	require.NoError(t, err)
	matches, err = regexp.MatchString(`^[a-zA-Z0-9]{256}$`, str4)
	require.NoError(t, err)
	assert.True(t, matches)

	assert.NotEqual(t, str3, str4)
}

func Test_RandomBytes(t *testing.T) {
	bytes1 := util.CryptoRandomBytes(32)
	bytes2 := util.CryptoRandomBytes(32)

	assert.Len(t, bytes1, 32)
	assert.Len(t, bytes2, 32)
	assert.NotEqual(t, bytes1, bytes2)

	bytes3 := util.CryptoRandomBytes(256)
	bytes4 := util.CryptoRandomBytes(256)

	assert.Len(t, bytes3, 256)
	assert.Len(t, bytes4, 256)
	assert.NotEqual(t, bytes3, bytes4)
}

// Test case for any function which accepts and returns a single string.
type StringTest struct {
	in, out string
}

var upperTests = []StringTest{
	{"", ""},
	{"ONLYUPPER", "ONLYUPPER"},
	{"abc", "ABC"},
	{"AbC123", "ABC123"},
	{"azAZ09_", "AZAZ09_"},
	{"longStrinGwitHmixofsmaLLandcAps", "LONGSTRINGWITHMIXOFSMALLANDCAPS"},
	{"long\u0250string\u0250with\u0250nonascii\u2C6Fchars", "LONG\u0250STRING\u0250WITH\u0250NONASCII\u2C6FCHARS"},
	{"\u0250\u0250\u0250\u0250\u0250", "\u0250\u0250\u0250\u0250\u0250"},
	{"a\u0080\U0010FFFF", "A\u0080\U0010FFFF"},
	{"lél", "LéL"},
}

func TestToUpperASCII(t *testing.T) {
	for _, tc := range upperTests {
		assert.Equal(t, util.ToUpperASCII(tc.in), tc.out)
	}
}

func BenchmarkToUpper(b *testing.B) {
	for _, tc := range upperTests {
		b.Run(tc.in, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				util.ToUpperASCII(tc.in)
			}
		})
	}
}

func TestToTitleCase(t *testing.T) {
	assert.Equal(t, `Foo Bar Baz`, util.ToTitleCase(`foo bar baz`))
	assert.Equal(t, `Foo Bar Baz`, util.ToTitleCase(`FOO BAR BAZ`))
}

func TestToPointer(t *testing.T) {
	assert.Equal(t, "abc", *util.ToPointer("abc"))
	assert.Equal(t, 123, *util.ToPointer(123))
	abc := "abc"
	assert.NotSame(t, &abc, util.ToPointer(abc))
	val123 := 123
	assert.NotSame(t, &val123, util.ToPointer(val123))
}

func TestReserveLineBreakForTextarea(t *testing.T) {
	assert.Equal(t, "test\ndata", util.ReserveLineBreakForTextarea("test\r\ndata"))
	assert.Equal(t, "test\ndata\n", util.ReserveLineBreakForTextarea("test\r\ndata\r\n"))
}

const (
	testPublicKey  = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAOhB7/zzhC+HXDdGOdLwJln5NYwm6UNXx3chmQSVTG4\n"
	testPrivateKey = `-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtz
c2gtZWQyNTUxOQAAACADoQe/884Qvh1w3RjnS8CZZ+TWMJulDV8d3IZkElUxuAAA
AIggISIjICEiIwAAAAtzc2gtZWQyNTUxOQAAACADoQe/884Qvh1w3RjnS8CZZ+TW
MJulDV8d3IZkElUxuAAAAEAAAQIDBAUGBwgJCgsMDQ4PEBESExQVFhcYGRobHB0e
HwOhB7/zzhC+HXDdGOdLwJln5NYwm6UNXx3chmQSVTG4AAAAAAECAwQF
-----END OPENSSH PRIVATE KEY-----` + "\n"
)

func TestGeneratingEd25519Keypair(t *testing.T) {
	defer test.MockProtect(&rand.Reader)()

	// Only 32 bytes needs to be provided to generate a ed25519 keypair.
	// And another 32 bytes are required, which is included as random value
	// in the OpenSSH format.
	b := make([]byte, 64)
	for i := 0; i < 64; i++ {
		b[i] = byte(i)
	}
	rand.Reader = bytes.NewReader(b)

	publicKey, privateKey, err := util.GenerateSSHKeypair()
	require.NoError(t, err)
	assert.Equal(t, testPublicKey, string(publicKey))
	assert.Equal(t, testPrivateKey, string(privateKey))
}

func TestOptionalArg(t *testing.T) {
	foo := func(other any, optArg ...int) int {
		return util.OptionalArg(optArg)
	}
	bar := func(other any, optArg ...int) int {
		return util.OptionalArg(optArg, 42)
	}
	assert.Equal(t, 0, foo(nil))
	assert.Equal(t, 100, foo(nil, 100))
	assert.Equal(t, 42, bar(nil))
	assert.Equal(t, 100, bar(nil, 100))
}
