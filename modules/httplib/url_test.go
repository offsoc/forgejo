// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package httplib

import (
	"testing"

	"forgejo.org/modules/setting"
	"forgejo.org/modules/test"

	"github.com/stretchr/testify/assert"
)

func TestIsRiskyRedirectURL(t *testing.T) {
	defer test.MockVariableValue(&setting.AppURL, "http://localhost:3000/sub/")()
	defer test.MockVariableValue(&setting.AppSubURL, "/sub")()

	tests := []struct {
		input string
		want  bool
	}{
		{"", false},
		{"foo", false},
		{"./", false},
		{"?key=val", false},
		{"/sub/", false},
		{"http://localhost:3000/sub/", false},
		{"/sub/foo", false},
		{"http://localhost:3000/sub/foo", false},
		{"http://localhost:3000/sub/test?param=false", false},
		// FIXME: should probably be true (would requires resolving references using setting.appURL.ResolveReference(u))
		{"/sub/../", false},
		{"http://localhost:3000/sub/../", false},
		{"/sUb/", false},
		{"http://localhost:3000/sUb/foo", false},
		{"/sub", false},
		{"/foo?k=%20#abc", false},
		{"/", false},
		{"a/", false},
		{"test?param=false", false},
		{"/hey/hey/hey#3244", false},

		{"//", true},
		{"\\\\", true},
		{"/\\", true},
		{"\\/", true},
		{"mail:a@b.com", true},
		{"https://test.com", true},
		{"http://localhost:3000/foo", true},
		{"http://localhost:3000/sub", true},
		{"http://localhost:3000/sub?key=val", true},
		{"https://example.com/", true},
		{"//example.com", true},
		{"http://example.com", true},
		{"http://localhost:3000/test?param=false", true},
		{"//localhost:3000/test?param=false", true},
		{"://missing protocol scheme", true},
		// FIXME: should probably be false
		{"//localhost:3000/sub/test?param=false", true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, IsRiskyRedirectURL(tt.input))
		})
	}
}

func TestIsRiskyRedirectURLWithoutSubURL(t *testing.T) {
	defer test.MockVariableValue(&setting.AppURL, "https://next.forgejo.org/")()
	defer test.MockVariableValue(&setting.AppSubURL, "")()

	tests := []struct {
		input string
		want  bool
	}{
		{"", false},
		{"foo", false},
		{"./", false},
		{"?key=val", false},
		{"/sub/", false},
		{"https://next.forgejo.org/sub/", false},
		{"/sub/foo", false},
		{"https://next.forgejo.org/sub/foo", false},
		{"https://next.forgejo.org/sub/test?param=false", false},
		{"https://next.forgejo.org/sub/../", false},
		{"/sub/../", false},
		{"/sUb/", false},
		{"https://next.forgejo.org/sUb/foo", false},
		{"/sub", false},
		{"/foo?k=%20#abc", false},
		{"/", false},
		{"a/", false},
		{"test?param=false", false},
		{"/hey/hey/hey#3244", false},
		{"https://next.forgejo.org/test?param=false", false},
		{"https://next.forgejo.org/foo", false},
		{"https://next.forgejo.org/sub", false},
		{"https://next.forgejo.org/sub?key=val", false},

		{"//", true},
		{"\\\\", true},
		{"/\\", true},
		{"\\/", true},
		{"mail:a@b.com", true},
		{"https://test.com", true},
		{"https://example.com/", true},
		{"//example.com", true},
		{"http://example.com", true},
		{"://missing protocol scheme", true},
		{"https://forgejo.org", true},
		{"https://example.org?url=https://next.forgejo.org", true},
		// FIXME: should probably be false
		{"https://next.forgejo.org", true},
		{"//next.forgejo.org/test?param=false", true},
		{"//next.forgejo.org/sub/test?param=false", true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, IsRiskyRedirectURL(tt.input))
		})
	}
}
