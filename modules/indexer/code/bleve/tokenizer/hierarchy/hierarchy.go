// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package hierarchy

import (
	"bytes"

	"github.com/blevesearch/bleve/v2/analysis"
	"github.com/blevesearch/bleve/v2/registry"
)

const Name = "path_hierarchy"

type PathHierarchyTokenizer struct{}

// Similar to elastic's path_hierarchy tokenizer
// This tokenizes a given path into all the possible hierarchies
// For example,
// modules/indexer/code/search.go =>
//
//	modules/
//	modules/indexer
//	modules/indexer/code
//	modules/indexer/code/search.go
func (t *PathHierarchyTokenizer) Tokenize(input []byte) analysis.TokenStream {
	// trim any extra slashes
	input = bytes.Trim(input, "/")

	// zero allocations until the nested directories exceed a depth of 8 (which is unlikely)
	rv := make(analysis.TokenStream, 0, 8)
	count, off := 1, 0

	// iterate till all directory separators
	for i := bytes.IndexRune(input[off:], '/'); i != -1; i = bytes.IndexRune(input[off:], '/') {
		// the index is relative to input[offset...]
		// add this index to the accumulated offset to get the index of the current separator in input[0...]
		off += i
		rv = append(rv, &analysis.Token{
			Term:     input[:off], // take the slice, input[0...index of separator]
			Start:    0,
			End:      off,
			Position: count,
			Type:     analysis.AlphaNumeric,
		})
		// increment the offset after considering the separator
		off++
		count++
	}

	// the entire file path should always be the last token
	rv = append(rv, &analysis.Token{
		Term:     input,
		Start:    0,
		End:      len(input),
		Position: count,
		Type:     analysis.AlphaNumeric,
	})

	return rv
}

func TokenizerConstructor(config map[string]any, cache *registry.Cache) (analysis.Tokenizer, error) {
	return &PathHierarchyTokenizer{}, nil
}

func init() {
	if err := registry.RegisterTokenizer(Name, TokenizerConstructor); err != nil {
		panic(err)
	}
}
