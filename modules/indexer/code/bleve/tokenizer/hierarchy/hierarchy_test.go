// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package hierarchy

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIndexerBleveHierarchyTokenizer(t *testing.T) {
	tokenizer := &PathHierarchyTokenizer{}
	keywords := []struct {
		Term    string
		Results []string
	}{
		{
			Term: "modules/indexer/code/search.go",
			Results: []string{
				"modules",
				"modules/indexer",
				"modules/indexer/code",
				"modules/indexer/code/search.go",
			},
		},
		{
			Term: "/tmp/forgejo/",
			Results: []string{
				"tmp",
				"tmp/forgejo",
			},
		},
		{
			Term: "a/b/c/d/e/f/g/h/i/j",
			Results: []string{
				"a",
				"a/b",
				"a/b/c",
				"a/b/c/d",
				"a/b/c/d/e",
				"a/b/c/d/e/f",
				"a/b/c/d/e/f/g",
				"a/b/c/d/e/f/g/h",
				"a/b/c/d/e/f/g/h/i",
				"a/b/c/d/e/f/g/h/i/j",
			},
		},
	}

	for _, kw := range keywords {
		tokens := tokenizer.Tokenize([]byte(kw.Term))
		assert.Len(t, tokens, len(kw.Results))
		for i, token := range tokens {
			assert.Equal(t, i+1, token.Position)
			assert.Equal(t, kw.Results[i], string(token.Term))
		}
	}
}
