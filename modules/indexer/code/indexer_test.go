// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package code

import (
	"os"
	"testing"

	"forgejo.org/models/db"
	"forgejo.org/models/unittest"
	"forgejo.org/modules/git"
	"forgejo.org/modules/indexer/code/bleve"
	"forgejo.org/modules/indexer/code/elasticsearch"
	"forgejo.org/modules/indexer/code/internal"

	_ "forgejo.org/models"
	_ "forgejo.org/models/actions"
	_ "forgejo.org/models/activities"
	_ "forgejo.org/models/forgefed"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	unittest.MainTest(m)
}

func testIndexer(name string, t *testing.T, indexer internal.Indexer) {
	t.Run(name, func(t *testing.T) {
		var repoID int64 = 1
		err := index(git.DefaultContext, indexer, repoID)
		require.NoError(t, err)
		keywords := []struct {
			RepoIDs  []int64
			Keyword  string
			IDs      []int64
			Langs    int
			Filename string
		}{
			{
				RepoIDs: nil,
				Keyword: "Description",
				IDs:     []int64{repoID},
				Langs:   1,
			},
			{
				RepoIDs: []int64{2},
				Keyword: "Description",
				IDs:     []int64{},
				Langs:   0,
			},
			{
				RepoIDs:  nil,
				Keyword:  "Description",
				IDs:      []int64{},
				Langs:    0,
				Filename: "NOT-README.md",
			},
			{
				RepoIDs:  nil,
				Keyword:  "Description",
				IDs:      []int64{repoID},
				Langs:    1,
				Filename: "README.md",
			},
			{
				RepoIDs: nil,
				Keyword: "Description for",
				IDs:     []int64{repoID},
				Langs:   1,
			},
			{
				RepoIDs: nil,
				Keyword: "repo1",
				IDs:     []int64{repoID},
				Langs:   1,
			},
			{
				RepoIDs: []int64{2},
				Keyword: "repo1",
				IDs:     []int64{},
				Langs:   0,
			},
			{
				RepoIDs: nil,
				Keyword: "non-exist",
				IDs:     []int64{},
				Langs:   0,
			},
		}

		for _, kw := range keywords {
			t.Run(kw.Keyword, func(t *testing.T) {
				total, res, langs, err := indexer.Search(t.Context(), &internal.SearchOptions{
					RepoIDs: kw.RepoIDs,
					Keyword: kw.Keyword,
					Paginator: &db.ListOptions{
						Page:     1,
						PageSize: 10,
					},
					Filename: kw.Filename,
					Mode:     SearchModeUnion,
				})
				require.NoError(t, err)
				assert.Len(t, kw.IDs, int(total))
				assert.Len(t, langs, kw.Langs)

				ids := make([]int64, 0, len(res))
				for _, hit := range res {
					ids = append(ids, hit.RepoID)
					assert.Equal(t, "# repo1\n\nDescription for repo1", hit.Content)
				}
				assert.Equal(t, kw.IDs, ids)
			})
		}

		require.NoError(t, indexer.Delete(t.Context(), repoID))
	})
}

func TestBleveIndexAndSearch(t *testing.T) {
	unittest.PrepareTestEnv(t)

	dir := t.TempDir()

	idx := bleve.NewIndexer(dir)
	_, err := idx.Init(t.Context())
	if err != nil {
		if idx != nil {
			idx.Close()
		}
		require.NoError(t, err)
	}
	defer idx.Close()

	testIndexer("bleve", t, idx)
}

func TestESIndexAndSearch(t *testing.T) {
	unittest.PrepareTestEnv(t)

	u := os.Getenv("TEST_INDEXER_CODE_ES_URL")
	if u == "" {
		t.SkipNow()
		return
	}

	indexer := elasticsearch.NewIndexer(u, "gitea_codes")
	if _, err := indexer.Init(t.Context()); err != nil {
		if indexer != nil {
			indexer.Close()
		}
		assert.FailNow(t, "Unable to init ES indexer", "error: %v", err)
	}

	defer indexer.Close()

	testIndexer("elastic_search", t, indexer)
}
