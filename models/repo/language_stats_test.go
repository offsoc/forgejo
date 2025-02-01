// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package repo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLanguagePercentages(t *testing.T) {
	testCases := []struct {
		input  LanguageStatList
		output map[string]float32
	}{
		{
			[]*LanguageStat{{Language: "Go", Size: 500}, {Language: "Rust", Size: 501}},
			map[string]float32{
				"Go":   50.0,
				"Rust": 50.0,
			},
		},
		{
			[]*LanguageStat{{Language: "Go", Size: 10}, {Language: "Rust", Size: 91}},
			map[string]float32{
				"Go":   9.9,
				"Rust": 90.1,
			},
		},
		{
			[]*LanguageStat{{Language: "Go", Size: 1}, {Language: "Rust", Size: 2}},
			map[string]float32{
				"Go":   33.3,
				"Rust": 66.7,
			},
		},
		{
			[]*LanguageStat{{Language: "Go", Size: 1}, {Language: "Rust", Size: 2}, {Language: "Shell", Size: 3}, {Language: "C#", Size: 4}, {Language: "Zig", Size: 5}, {Language: "Coq", Size: 6}, {Language: "Haskell", Size: 7}},
			map[string]float32{
				"Go":      3.6,
				"Rust":    7.1,
				"Shell":   10.7,
				"C#":      14.3,
				"Zig":     17.9,
				"Coq":     21.4,
				"Haskell": 25,
			},
		},
		{
			[]*LanguageStat{{Language: "Go", Size: 1000}, {Language: "PHP", Size: 1}, {Language: "Java", Size: 1}},
			map[string]float32{
				"Go":    99.8,
				"other": 0.2,
			},
		},
		{
			[]*LanguageStat{},
			map[string]float32{},
		},
	}

	for _, testCase := range testCases {
		assert.Equal(t, testCase.output, testCase.input.getLanguagePercentages())
	}
}
