// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package elasticsearch

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"forgejo.org/modules/graceful"
	indexer_internal "forgejo.org/modules/indexer/internal"
	inner_elasticsearch "forgejo.org/modules/indexer/internal/elasticsearch"
	"forgejo.org/modules/indexer/issues/internal"

	"github.com/olivere/elastic/v7"
)

const (
	issueIndexerLatestVersion = 2
	// multi-match-types, currently only 2 types are used
	// Reference: https://www.elastic.co/guide/en/elasticsearch/reference/7.0/query-dsl-multi-match-query.html#multi-match-types
	esMultiMatchTypeBestFields   = "best_fields"
	esMultiMatchTypePhrasePrefix = "phrase_prefix"

	// fuzziness options
	// Reference: https://www.elastic.co/guide/en/elasticsearch/reference/7.0/common-options.html#fuzziness
	esFuzzyAuto = "AUTO"
)

var _ internal.Indexer = &Indexer{}

// Indexer implements Indexer interface
type Indexer struct {
	inner                    *inner_elasticsearch.Indexer
	indexer_internal.Indexer // do not composite inner_elasticsearch.Indexer directly to avoid exposing too much
}

// NewIndexer creates a new elasticsearch indexer
func NewIndexer(url, indexerName string) *Indexer {
	inner := inner_elasticsearch.NewIndexer(url, indexerName, issueIndexerLatestVersion, defaultMapping)
	indexer := &Indexer{
		inner:   inner,
		Indexer: inner,
	}
	return indexer
}

const (
	defaultMapping = `
{
	"mappings": {
		"properties": {
			"id": { "type": "long", "index": true },
			"repo_id": { "type": "long", "index": true },
			"is_public": { "type": "boolean", "index": true },

			"index": { "type": "long", "index": true },
			"title": { "type": "text", "index": true },
			"content": { "type": "text", "index": true },
			"comments": { "type" : "text", "index": true },

			"is_pull": { "type": "boolean", "index": true },
			"is_closed": { "type": "boolean", "index": true },
			"label_ids": { "type": "long", "index": true },
			"no_label": { "type": "boolean", "index": true },
			"milestone_id": { "type": "long", "index": true },
			"project_id": { "type": "long", "index": true },
			"project_board_id": { "type": "long", "index": true },
			"poster_id": { "type": "long", "index": true },
			"assignee_id": { "type": "long", "index": true },
			"mention_ids": { "type": "long", "index": true },
			"reviewed_ids": { "type": "long", "index": true },
			"review_requested_ids": { "type": "long", "index": true },
			"subscriber_ids": { "type": "long", "index": true },
			"updated_unix": { "type": "long", "index": true },

			"created_unix": { "type": "long", "index": true },
			"deadline_unix": { "type": "long", "index": true },
			"comment_count": { "type": "long", "index": true }
		}
	}
}
`
)

// Index will save the index data
func (b *Indexer) Index(ctx context.Context, issues ...*internal.IndexerData) error {
	if len(issues) == 0 {
		return nil
	} else if len(issues) == 1 {
		issue := issues[0]
		_, err := b.inner.Client.Index().
			Index(b.inner.VersionedIndexName()).
			Id(fmt.Sprintf("%d", issue.ID)).
			BodyJson(issue).
			Do(ctx)
		return err
	}

	reqs := make([]elastic.BulkableRequest, 0)
	for _, issue := range issues {
		reqs = append(reqs,
			elastic.NewBulkIndexRequest().
				Index(b.inner.VersionedIndexName()).
				Id(fmt.Sprintf("%d", issue.ID)).
				Doc(issue),
		)
	}

	_, err := b.inner.Client.Bulk().
		Index(b.inner.VersionedIndexName()).
		Add(reqs...).
		Do(graceful.GetManager().HammerContext())
	return err
}

// Delete deletes indexes by ids
func (b *Indexer) Delete(ctx context.Context, ids ...int64) error {
	if len(ids) == 0 {
		return nil
	} else if len(ids) == 1 {
		_, err := b.inner.Client.Delete().
			Index(b.inner.VersionedIndexName()).
			Id(fmt.Sprintf("%d", ids[0])).
			Do(ctx)
		return err
	}

	reqs := make([]elastic.BulkableRequest, 0)
	for _, id := range ids {
		reqs = append(reqs,
			elastic.NewBulkDeleteRequest().
				Index(b.inner.VersionedIndexName()).
				Id(fmt.Sprintf("%d", id)),
		)
	}

	_, err := b.inner.Client.Bulk().
		Index(b.inner.VersionedIndexName()).
		Add(reqs...).
		Do(graceful.GetManager().HammerContext())
	return err
}

// Search searches for issues by given conditions.
// Returns the matching issue IDs
func (b *Indexer) Search(ctx context.Context, options *internal.SearchOptions) (*internal.SearchResult, error) {
	query := elastic.NewBoolQuery()

	if options.Keyword != "" {
		q := elastic.NewBoolQuery()
		tokens, err := options.Tokens()
		if err != nil {
			return nil, err
		}
		for _, token := range tokens {
			innerQ := elastic.NewMultiMatchQuery(token.Term, "content", "comments").FieldWithBoost("title", 2.0).TieBreaker(0.5)
			if token.Fuzzy {
				// If the term is not a phrase use fuzziness set to AUTO
				innerQ = innerQ.Type(esMultiMatchTypeBestFields).Fuzziness(esFuzzyAuto)
			} else {
				innerQ = innerQ.Type(esMultiMatchTypePhrasePrefix)
			}
			var eitherQ elastic.Query = innerQ
			if issueID, err := token.ParseIssueReference(); err == nil {
				indexQ := elastic.NewTermQuery("index", issueID).Boost(15.0)
				eitherQ = elastic.NewDisMaxQuery().Query(indexQ).Query(innerQ).TieBreaker(0.5)
			}
			switch token.Kind {
			case internal.BoolOptMust:
				q.Must(eitherQ)
			case internal.BoolOptShould:
				q.Should(eitherQ)
			case internal.BoolOptNot:
				q.MustNot(eitherQ)
			}
		}
		query.Must(q)
	}

	if len(options.RepoIDs) > 0 {
		q := elastic.NewBoolQuery()
		q.Should(elastic.NewTermsQuery("repo_id", toAnySlice(options.RepoIDs)...))
		if options.AllPublic {
			q.Should(elastic.NewTermQuery("is_public", true))
		}
		query.Must(q)
	}

	if options.IsPull.Has() {
		query.Must(elastic.NewTermQuery("is_pull", options.IsPull.Value()))
	}
	if options.IsClosed.Has() {
		query.Must(elastic.NewTermQuery("is_closed", options.IsClosed.Value()))
	}

	if options.NoLabelOnly {
		query.Must(elastic.NewTermQuery("no_label", true))
	} else {
		if len(options.IncludedLabelIDs) > 0 {
			q := elastic.NewBoolQuery()
			for _, labelID := range options.IncludedLabelIDs {
				q.Must(elastic.NewTermQuery("label_ids", labelID))
			}
			query.Must(q)
		} else if len(options.IncludedAnyLabelIDs) > 0 {
			query.Must(elastic.NewTermsQuery("label_ids", toAnySlice(options.IncludedAnyLabelIDs)...))
		}
		if len(options.ExcludedLabelIDs) > 0 {
			q := elastic.NewBoolQuery()
			for _, labelID := range options.ExcludedLabelIDs {
				q.MustNot(elastic.NewTermQuery("label_ids", labelID))
			}
			query.Must(q)
		}
	}

	if len(options.MilestoneIDs) > 0 {
		query.Must(elastic.NewTermsQuery("milestone_id", toAnySlice(options.MilestoneIDs)...))
	}

	if options.ProjectID.Has() {
		query.Must(elastic.NewTermQuery("project_id", options.ProjectID.Value()))
	}
	if options.ProjectColumnID.Has() {
		query.Must(elastic.NewTermQuery("project_board_id", options.ProjectColumnID.Value()))
	}

	if options.PosterID.Has() {
		query.Must(elastic.NewTermQuery("poster_id", options.PosterID.Value()))
	}

	if options.AssigneeID.Has() {
		query.Must(elastic.NewTermQuery("assignee_id", options.AssigneeID.Value()))
	}

	if options.MentionID.Has() {
		query.Must(elastic.NewTermQuery("mention_ids", options.MentionID.Value()))
	}

	if options.ReviewedID.Has() {
		query.Must(elastic.NewTermQuery("reviewed_ids", options.ReviewedID.Value()))
	}
	if options.ReviewRequestedID.Has() {
		query.Must(elastic.NewTermQuery("review_requested_ids", options.ReviewRequestedID.Value()))
	}

	if options.SubscriberID.Has() {
		query.Must(elastic.NewTermQuery("subscriber_ids", options.SubscriberID.Value()))
	}

	if options.UpdatedAfterUnix.Has() || options.UpdatedBeforeUnix.Has() {
		q := elastic.NewRangeQuery("updated_unix")
		if options.UpdatedAfterUnix.Has() {
			q.Gte(options.UpdatedAfterUnix.Value())
		}
		if options.UpdatedBeforeUnix.Has() {
			q.Lte(options.UpdatedBeforeUnix.Value())
		}
		query.Must(q)
	}

	if options.SortBy == "" {
		options.SortBy = internal.SortByScore
	}
	sortBy := []elastic.Sorter{
		parseSortBy(options.SortBy),
		elastic.NewFieldSort("id").Desc(),
	}

	// See https://stackoverflow.com/questions/35206409/elasticsearch-2-1-result-window-is-too-large-index-max-result-window/35221900
	// TODO: make it configurable since it's configurable in elasticsearch
	const maxPageSize = 10000

	skip, limit := indexer_internal.ParsePaginator(options.Paginator, maxPageSize)
	searchResult, err := b.inner.Client.Search().
		Index(b.inner.VersionedIndexName()).
		Query(query).
		SortBy(sortBy...).
		From(skip).Size(limit).
		Do(ctx)
	if err != nil {
		return nil, err
	}

	hits := make([]internal.Match, 0, limit)
	for _, hit := range searchResult.Hits.Hits {
		id, _ := strconv.ParseInt(hit.Id, 10, 64)
		hits = append(hits, internal.Match{
			ID: id,
		})
	}

	return &internal.SearchResult{
		Total: searchResult.TotalHits(),
		Hits:  hits,
	}, nil
}

func toAnySlice[T any](s []T) []any {
	ret := make([]any, 0, len(s))
	for _, item := range s {
		ret = append(ret, item)
	}
	return ret
}

func parseSortBy(sortBy internal.SortBy) elastic.Sorter {
	field := strings.TrimPrefix(string(sortBy), "-")
	ret := elastic.NewFieldSort(field)
	if strings.HasPrefix(string(sortBy), "-") {
		ret.Desc()
	}
	return ret
}
