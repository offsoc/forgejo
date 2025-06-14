// Copyright 2018 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package bleve

import (
	"context"

	indexer_internal "forgejo.org/modules/indexer/internal"
	inner_bleve "forgejo.org/modules/indexer/internal/bleve"
	"forgejo.org/modules/indexer/issues/internal"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/custom"
	"github.com/blevesearch/bleve/v2/analysis/token/camelcase"
	"github.com/blevesearch/bleve/v2/analysis/token/lowercase"
	"github.com/blevesearch/bleve/v2/analysis/token/unicodenorm"
	"github.com/blevesearch/bleve/v2/analysis/tokenizer/unicode"
	"github.com/blevesearch/bleve/v2/mapping"
	"github.com/blevesearch/bleve/v2/search/query"
)

const (
	issueIndexerAnalyzer      = "issueIndexer"
	issueIndexerDocType       = "issueIndexerDocType"
	issueIndexerLatestVersion = 5
)

const unicodeNormalizeName = "unicodeNormalize"

func addUnicodeNormalizeTokenFilter(m *mapping.IndexMappingImpl) error {
	return m.AddCustomTokenFilter(unicodeNormalizeName, map[string]any{
		"type": unicodenorm.Name,
		"form": unicodenorm.NFC,
	})
}

const maxBatchSize = 16

// IndexerData an update to the issue indexer
type IndexerData internal.IndexerData

// Type returns the document type, for bleve's mapping.Classifier interface.
func (i *IndexerData) Type() string {
	return issueIndexerDocType
}

// generateIssueIndexMapping generates the bleve index mapping for issues
func generateIssueIndexMapping() (mapping.IndexMapping, error) {
	mapping := bleve.NewIndexMapping()
	docMapping := bleve.NewDocumentMapping()

	numericFieldMapping := bleve.NewNumericFieldMapping()
	numericFieldMapping.Store = false
	numericFieldMapping.IncludeInAll = false
	docMapping.AddFieldMappingsAt("repo_id", numericFieldMapping)

	textFieldMapping := bleve.NewTextFieldMapping()
	textFieldMapping.Store = false
	textFieldMapping.IncludeInAll = false

	boolFieldMapping := bleve.NewBooleanFieldMapping()
	boolFieldMapping.Store = false
	boolFieldMapping.IncludeInAll = false

	numberFieldMapping := bleve.NewNumericFieldMapping()
	numberFieldMapping.Store = false
	numberFieldMapping.IncludeInAll = false

	docMapping.AddFieldMappingsAt("is_public", boolFieldMapping)

	docMapping.AddFieldMappingsAt("index", numberFieldMapping)
	docMapping.AddFieldMappingsAt("title", textFieldMapping)
	docMapping.AddFieldMappingsAt("content", textFieldMapping)
	docMapping.AddFieldMappingsAt("comments", textFieldMapping)

	docMapping.AddFieldMappingsAt("is_pull", boolFieldMapping)
	docMapping.AddFieldMappingsAt("is_closed", boolFieldMapping)
	docMapping.AddFieldMappingsAt("label_ids", numberFieldMapping)
	docMapping.AddFieldMappingsAt("no_label", boolFieldMapping)
	docMapping.AddFieldMappingsAt("milestone_id", numberFieldMapping)
	docMapping.AddFieldMappingsAt("project_id", numberFieldMapping)
	docMapping.AddFieldMappingsAt("project_board_id", numberFieldMapping)
	docMapping.AddFieldMappingsAt("poster_id", numberFieldMapping)
	docMapping.AddFieldMappingsAt("assignee_id", numberFieldMapping)
	docMapping.AddFieldMappingsAt("mention_ids", numberFieldMapping)
	docMapping.AddFieldMappingsAt("reviewed_ids", numberFieldMapping)
	docMapping.AddFieldMappingsAt("review_requested_ids", numberFieldMapping)
	docMapping.AddFieldMappingsAt("subscriber_ids", numberFieldMapping)
	docMapping.AddFieldMappingsAt("updated_unix", numberFieldMapping)

	docMapping.AddFieldMappingsAt("created_unix", numberFieldMapping)
	docMapping.AddFieldMappingsAt("deadline_unix", numberFieldMapping)
	docMapping.AddFieldMappingsAt("comment_count", numberFieldMapping)

	if err := addUnicodeNormalizeTokenFilter(mapping); err != nil {
		return nil, err
	} else if err = mapping.AddCustomAnalyzer(issueIndexerAnalyzer, map[string]any{
		"type":          custom.Name,
		"char_filters":  []string{},
		"tokenizer":     unicode.Name,
		"token_filters": []string{unicodeNormalizeName, camelcase.Name, lowercase.Name},
	}); err != nil {
		return nil, err
	}

	mapping.DefaultAnalyzer = issueIndexerAnalyzer
	mapping.AddDocumentMapping(issueIndexerDocType, docMapping)
	mapping.AddDocumentMapping("_all", bleve.NewDocumentDisabledMapping())
	mapping.DefaultMapping = bleve.NewDocumentDisabledMapping() // disable default mapping, avoid indexing unexpected structs

	return mapping, nil
}

var _ internal.Indexer = &Indexer{}

// Indexer implements Indexer interface
type Indexer struct {
	inner                    *inner_bleve.Indexer
	indexer_internal.Indexer // do not composite inner_bleve.Indexer directly to avoid exposing too much
}

// NewIndexer creates a new bleve local indexer
func NewIndexer(indexDir string) *Indexer {
	inner := inner_bleve.NewIndexer(indexDir, issueIndexerLatestVersion, generateIssueIndexMapping)
	return &Indexer{
		Indexer: inner,
		inner:   inner,
	}
}

// Index will save the index data
func (b *Indexer) Index(_ context.Context, issues ...*internal.IndexerData) error {
	batch := inner_bleve.NewFlushingBatch(b.inner.Indexer, maxBatchSize)
	for _, issue := range issues {
		if err := batch.Index(indexer_internal.Base36(issue.ID), (*IndexerData)(issue)); err != nil {
			return err
		}
	}
	return batch.Flush()
}

// Delete deletes indexes by ids
func (b *Indexer) Delete(_ context.Context, ids ...int64) error {
	batch := inner_bleve.NewFlushingBatch(b.inner.Indexer, maxBatchSize)
	for _, id := range ids {
		if err := batch.Delete(indexer_internal.Base36(id)); err != nil {
			return err
		}
	}
	return batch.Flush()
}

// Search searches for issues by given conditions.
// Returns the matching issue IDs
func (b *Indexer) Search(ctx context.Context, options *internal.SearchOptions) (*internal.SearchResult, error) {
	var queries []query.Query

	if options.Keyword != "" {
		tokens, err := options.Tokens()
		if err != nil {
			return nil, err
		}
		q := bleve.NewBooleanQuery()
		for _, token := range tokens {
			innerQ := bleve.NewDisjunctionQuery(
				inner_bleve.MatchPhraseQuery(token.Term, "title", issueIndexerAnalyzer, token.Fuzzy, 2.0),
				inner_bleve.MatchPhraseQuery(token.Term, "content", issueIndexerAnalyzer, token.Fuzzy, 1.0),
				inner_bleve.MatchPhraseQuery(token.Term, "comments", issueIndexerAnalyzer, token.Fuzzy, 1.0))

			if issueID, err := token.ParseIssueReference(); err == nil {
				idQuery := inner_bleve.NumericEqualityQuery(issueID, "index")
				idQuery.SetBoost(5.0)
				innerQ.AddQuery(idQuery)
			}

			switch token.Kind {
			case internal.BoolOptMust:
				q.AddMust(innerQ)
			case internal.BoolOptShould:
				q.AddShould(innerQ)
			case internal.BoolOptNot:
				q.AddMustNot(innerQ)
			}
		}
		queries = append(queries, q)
	}

	if len(options.RepoIDs) > 0 || options.AllPublic {
		var repoQueries []query.Query
		for _, repoID := range options.RepoIDs {
			repoQueries = append(repoQueries, inner_bleve.NumericEqualityQuery(repoID, "repo_id"))
		}
		if options.AllPublic {
			repoQueries = append(repoQueries, inner_bleve.BoolFieldQuery(true, "is_public"))
		}
		queries = append(queries, bleve.NewDisjunctionQuery(repoQueries...))
	}

	if options.IsPull.Has() {
		queries = append(queries, inner_bleve.BoolFieldQuery(options.IsPull.Value(), "is_pull"))
	}
	if options.IsClosed.Has() {
		queries = append(queries, inner_bleve.BoolFieldQuery(options.IsClosed.Value(), "is_closed"))
	}

	if options.NoLabelOnly {
		queries = append(queries, inner_bleve.BoolFieldQuery(true, "no_label"))
	} else {
		if len(options.IncludedLabelIDs) > 0 {
			var includeQueries []query.Query
			for _, labelID := range options.IncludedLabelIDs {
				includeQueries = append(includeQueries, inner_bleve.NumericEqualityQuery(labelID, "label_ids"))
			}
			queries = append(queries, bleve.NewConjunctionQuery(includeQueries...))
		} else if len(options.IncludedAnyLabelIDs) > 0 {
			var includeQueries []query.Query
			for _, labelID := range options.IncludedAnyLabelIDs {
				includeQueries = append(includeQueries, inner_bleve.NumericEqualityQuery(labelID, "label_ids"))
			}
			queries = append(queries, bleve.NewDisjunctionQuery(includeQueries...))
		}
		if len(options.ExcludedLabelIDs) > 0 {
			var excludeQueries []query.Query
			for _, labelID := range options.ExcludedLabelIDs {
				q := bleve.NewBooleanQuery()
				q.AddMustNot(inner_bleve.NumericEqualityQuery(labelID, "label_ids"))
				excludeQueries = append(excludeQueries, q)
			}
			queries = append(queries, bleve.NewConjunctionQuery(excludeQueries...))
		}
	}

	if len(options.MilestoneIDs) > 0 {
		var milestoneQueries []query.Query
		for _, milestoneID := range options.MilestoneIDs {
			milestoneQueries = append(milestoneQueries, inner_bleve.NumericEqualityQuery(milestoneID, "milestone_id"))
		}
		queries = append(queries, bleve.NewDisjunctionQuery(milestoneQueries...))
	}

	if options.ProjectID.Has() {
		queries = append(queries, inner_bleve.NumericEqualityQuery(options.ProjectID.Value(), "project_id"))
	}
	if options.ProjectColumnID.Has() {
		queries = append(queries, inner_bleve.NumericEqualityQuery(options.ProjectColumnID.Value(), "project_board_id"))
	}

	if options.PosterID.Has() {
		queries = append(queries, inner_bleve.NumericEqualityQuery(options.PosterID.Value(), "poster_id"))
	}

	if options.AssigneeID.Has() {
		queries = append(queries, inner_bleve.NumericEqualityQuery(options.AssigneeID.Value(), "assignee_id"))
	}

	if options.MentionID.Has() {
		queries = append(queries, inner_bleve.NumericEqualityQuery(options.MentionID.Value(), "mention_ids"))
	}

	if options.ReviewedID.Has() {
		queries = append(queries, inner_bleve.NumericEqualityQuery(options.ReviewedID.Value(), "reviewed_ids"))
	}
	if options.ReviewRequestedID.Has() {
		queries = append(queries, inner_bleve.NumericEqualityQuery(options.ReviewRequestedID.Value(), "review_requested_ids"))
	}

	if options.SubscriberID.Has() {
		queries = append(queries, inner_bleve.NumericEqualityQuery(options.SubscriberID.Value(), "subscriber_ids"))
	}

	if options.UpdatedAfterUnix.Has() || options.UpdatedBeforeUnix.Has() {
		queries = append(queries, inner_bleve.NumericRangeInclusiveQuery(
			options.UpdatedAfterUnix,
			options.UpdatedBeforeUnix,
			"updated_unix"))
	}

	var indexerQuery query.Query = bleve.NewConjunctionQuery(queries...)
	if len(queries) == 0 {
		indexerQuery = bleve.NewMatchAllQuery()
	}

	skip, limit := indexer_internal.ParsePaginator(options.Paginator)
	search := bleve.NewSearchRequestOptions(indexerQuery, limit, skip, false)

	if options.SortBy == "" {
		options.SortBy = internal.SortByCreatedAsc
	}

	search.SortBy([]string{string(options.SortBy), "-_id"})

	result, err := b.inner.Indexer.SearchInContext(ctx, search)
	if err != nil {
		return nil, err
	}

	ret := &internal.SearchResult{
		Total: int64(result.Total),
		Hits:  make([]internal.Match, 0, len(result.Hits)),
	}
	for _, hit := range result.Hits {
		id, err := indexer_internal.ParseBase36(hit.ID)
		if err != nil {
			return nil, err
		}
		ret.Hits = append(ret.Hits, internal.Match{
			ID: id,
		})
	}
	return ret, nil
}
