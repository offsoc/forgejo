// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT
package integration

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"forgejo.org/services/repository/files"
)

func TestSanitizePath(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    string
		expectError bool
	}{
		// Valid paths
		{
			name:     "simple valid path",
			input:    "folder/file.txt",
			expected: "folder/file.txt",
		},
		{
			name:     "single file",
			input:    "file.txt",
			expected: "file.txt",
		},
		{
			name:     "nested path",
			input:    "a/b/c/file.txt",
			expected: "a/b/c/file.txt",
		},

		// Path normalization
		{
			name:     "backslash to forward slash",
			input:    "folder\\file.txt",
			expected: "folder/file.txt",
		},
		{
			name:     "mixed separators",
			input:    "folder\\subfolder/file.txt",
			expected: "folder/subfolder/file.txt",
		},
		{
			name:     "double separators",
			input:    "folder//file.txt",
			expected: "folder/file.txt",
		},
		{
			name:     "trailing slash",
			input:    "folder/file.txt/",
			expected: "folder/file.txt",
		},
		{
			name:     "dot segments",
			input:    "folder/./file.txt",
			expected: "folder/file.txt",
		},
		{
			name:     "parent directory references",
			input:    "folder/../other/file.txt",
			expected: "other/file.txt",
		},

		// Character sanitization
		{
			name:     "illegal characters replaced",
			input:    "file<name>.txt",
			expected: "file_name_.txt",
		},
		{
			name:     "multiple illegal characters",
			input:    "file:name|with?bad*chars.txt",
			expected: "file_name_with_bad_chars.txt",
		},
		{
			name:     "quotes in filename",
			input:    `file"name.txt`,
			expected: "file_name.txt",
		},
		{
			name:     "control characters",
			input:    "file\x00\x01name.txt",
			expected: "file__name.txt",
		},

		// Whitespace handling
		{
			name:     "leading whitespace",
			input:    " file.txt",
			expected: "file.txt",
		},
		{
			name:     "trailing whitespace",
			input:    "file.txt ",
			expected: "file.txt",
		},
		{
			name:     "whitespace in path components",
			input:    " folder / file.txt ",
			expected: "folder/file.txt",
		},

		// Edge cases that should return errors
		{
			name:        "path starts with slash",
			input:       "/folder/file.txt",
			expectError: true,
		},
		{
			name:        "empty string",
			input:       "",
			expectError: true,
		},
		{
			name:        "only separators",
			input:       "///",
			expectError: true,
		},
		{
			name:     "only illegal characters",
			input:    "<>:\"|?*",
			expected: "_______",
		},
		{
			name:        "only whitespace",
			input:       "   ",
			expectError: true,
		},
		{
			name:        "path that resolves to root",
			input:       "../..",
			expectError: true,
		},
		{
			name:        "path that goes above root",
			input:       "folder/../../..",
			expectError: true,
		},

		// Complex combinations
		{
			name:     "complex path with multiple issues",
			input:    "folder\\with:illegal|chars/normal_file.txt",
			expected: "folder/with_illegal_chars/normal_file.txt",
		},
		{
			name:     "unicode characters preserved",
			input:    "folder/файл.txt",
			expected: "folder/файл.txt",
		},
		{
			name:     "dots and extensions",
			input:    "file.name.with.dots.txt",
			expected: "file.name.with.dots.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := files.SanitizePath(tt.input)

			if tt.expectError {
				require.Error(t, err, "expected error for input %q", tt.input)
				return
			}

			require.NoError(t, err, "unexpected error for input %q", tt.input)
			assert.Equal(t, tt.expected, result, "SanitizePath(%q) should return expected result", tt.input)
		})
	}
}

// TestSanitizePathErrorMessages tests that error messages are informative
func TestSanitizePathErrorMessages(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedError string
	}{
		{
			name:          "path starts with slash",
			input:         "/test/path",
			expectedError: "path starts with / : /test/path",
		},
		{
			name:          "path that resolves to root",
			input:         "../..",
			expectedError: "path resolves to root or becomes empty after cleaning",
		},
		{
			name:          "empty after sanitization",
			input:         "",
			expectedError: "path resolves to root or becomes empty after cleaning",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := files.SanitizePath(tt.input)
			require.Error(t, err, "expected error for input %q", tt.input)
			assert.Equal(t, tt.expectedError, err.Error(), "error message for %q should match expected", tt.input)
		})
	}
}
