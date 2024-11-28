// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package markdown

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatchColor(t *testing.T) {
	testCases := []struct {
		input    string
		expected bool
	}{
		{"#ddeeffa0", true},
		{"#ddeefe", true},
		{"#abcdef", true},
		{"#fffa", true},
		{"#abcdeg", false},
		{"#abcdefg0", false},
		{"black", false},
		{"violet", false},
		{"rgb(255, 255, 255)", true},
		{"rgb(0, 0, 0)", true},
		{"rgb(256, 0, 0)", false},
		{"rgb(0, 256, 0)", false},
		{"rgb(0, 0, 256)", false},
		{"rgb(0, 0, 0, 1)", false},
		{"rgba(0, 0, 0)", false},
		{"rgba(0, 255, 0, 1)", true},
		{"rgba(32, 255, 12, 0.55)", true},
		{"rgba(32, 256, 12, 0.55)", false},
		{"hsl(0, 0%, 0%)", true},
		{"hsl(360, 100%, 100%)", true},
		{"hsl(361, 100%, 50%)", false},
		{"hsl(360, 101%, 50%)", false},
		{"hsl(360, 100%, 101%)", false},
		{"hsl(0, 0%, 0%, 0)", false},
		{"hsla(0, 0%, 0%)", false},
		{"hsla(0, 0%, 0%, 0)", true},
		{"hsla(0, 0%, 0%, 1)", true},
		{"hsla(0, 0%, 0%, 0.5)", true},
		{"hsla(0, 0%, 0%, 1.5)", false},
	}
	for _, testCase := range testCases {
		actual := matchColor(testCase.input)
		assert.Equal(t, testCase.expected, actual)
	}
}
