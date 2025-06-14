// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package bleve

import (
	"forgejo.org/modules/optional"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search/query"
)

// NumericEqualityQuery generates a numeric equality query for the given value and field
func NumericEqualityQuery(value int64, field string) *query.NumericRangeQuery {
	f := float64(value)
	tru := true                                                  // codespell:ignore
	q := bleve.NewNumericRangeInclusiveQuery(&f, &f, &tru, &tru) // codespell:ignore
	q.SetField(field)
	return q
}

// MatchQuery generates a match query for the given phrase, field and analyzer
func MatchQuery(matchTerm, field, analyzer string, fuzziness int) *query.MatchQuery {
	q := bleve.NewMatchQuery(matchTerm)
	q.FieldVal = field
	q.Analyzer = analyzer
	q.Fuzziness = fuzziness
	return q
}

// MatchPhraseQuery generates a match phrase query for the given phrase, field and analyzer
func MatchPhraseQuery(matchPhrase, field, analyzer string, autoFuzzy bool, boost float64) *query.MatchPhraseQuery {
	q := bleve.NewMatchPhraseQuery(matchPhrase)
	q.FieldVal = field
	q.Analyzer = analyzer
	q.SetAutoFuzziness(autoFuzzy)
	q.SetBoost(boost)
	return q
}

// BoolFieldQuery generates a bool field query for the given value and field
func BoolFieldQuery(value bool, field string) *query.BoolFieldQuery {
	q := bleve.NewBoolFieldQuery(value)
	q.SetField(field)
	return q
}

func NumericRangeInclusiveQuery(min, max optional.Option[int64], field string) *query.NumericRangeQuery {
	var minF, maxF *float64
	var minI, maxI *bool
	if min.Has() {
		minF = new(float64)
		*minF = float64(min.Value())
		minI = new(bool)
		*minI = true
	}
	if max.Has() {
		maxF = new(float64)
		*maxF = float64(max.Value())
		maxI = new(bool)
		*maxI = true
	}
	q := bleve.NewNumericRangeInclusiveQuery(minF, maxF, minI, maxI)
	q.SetField(field)
	return q
}
