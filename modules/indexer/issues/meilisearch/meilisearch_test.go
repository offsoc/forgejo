// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package meilisearch

import (
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"forgejo.org/modules/indexer/issues/internal"
	"forgejo.org/modules/indexer/issues/internal/tests"

	"github.com/meilisearch/meilisearch-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMeilisearchIndexer(t *testing.T) {
	t.Skip("meilisearch not found in Forgejo test yet")
	// The meilisearch instance started by pull-db-tests.yml > test-unit > services > meilisearch
	url := "http://meilisearch:7700"
	key := "" // auth has been disabled in test environment

	if os.Getenv("CI") == "" {
		// Make it possible to run tests against a local meilisearch instance
		url = os.Getenv("TEST_MEILISEARCH_URL")
		if url == "" {
			t.Skip("TEST_MEILISEARCH_URL not set and not running in CI")
			return
		}
		key = os.Getenv("TEST_MEILISEARCH_KEY")
	}

	require.Eventually(t, func() bool {
		resp, err := http.Get(url)
		return err == nil && resp.StatusCode == http.StatusOK
	}, time.Minute, time.Microsecond*100, "Failed to wait for meilisearch to be up")

	indexer := NewIndexer(url, key, fmt.Sprintf("test_meilisearch_indexer_%d", time.Now().Unix()))
	defer indexer.Close()

	tests.TestIndexer(t, indexer)
}

func TestConvertHits(t *testing.T) {
	_, err := convertHits(&meilisearch.SearchResponse{
		Hits: []any{"aa", "bb", "cc", "dd"},
	})
	require.ErrorIs(t, err, ErrMalformedResponse)

	validResponse := &meilisearch.SearchResponse{
		Hits: []any{
			map[string]any{
				"id":       float64(11),
				"title":    "a title",
				"content":  "issue body with no match",
				"comments": []any{"hey what's up?", "I'm currently bowling", "nice"},
			},
			map[string]any{
				"id":       float64(22),
				"title":    "Bowling as title",
				"content":  "",
				"comments": []any{},
			},
			map[string]any{
				"id":       float64(33),
				"title":    "Bowl-ing as fuzzy match",
				"content":  "",
				"comments": []any{},
			},
		},
	}
	hits, err := convertHits(validResponse)
	require.NoError(t, err)
	assert.Equal(t, []internal.Match{{ID: 11}, {ID: 22}, {ID: 33}}, hits)
}

func TestDoubleQuoteKeyword(t *testing.T) {
	assert.Empty(t, doubleQuoteKeyword(""))
	assert.Equal(t, `"a" "b" "c"`, doubleQuoteKeyword("a b c"))
	assert.Equal(t, `"a" "d" "g"`, doubleQuoteKeyword("a  d g"))
	assert.Equal(t, `"a" "d" "g"`, doubleQuoteKeyword("a  d g"))
	assert.Equal(t, `"a" "d" "g"`, doubleQuoteKeyword(`a  "" "d" """g`))
}
