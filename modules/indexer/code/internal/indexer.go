// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package internal

import (
	"context"
	"errors"

	"forgejo.org/models/db"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/modules/indexer/internal"
)

// Indexer defines an interface to index and search code contents
type Indexer interface {
	internal.Indexer
	Index(ctx context.Context, repo *repo_model.Repository, sha string, changes *RepoChanges) error
	Delete(ctx context.Context, repoID int64) error
	Search(ctx context.Context, opts *SearchOptions) (int64, []*SearchResult, []*SearchResultLanguages, error)
}

type CodeSearchMode int

const (
	CodeSearchModeExact CodeSearchMode = iota
	CodeSearchModeUnion
)

func (mode CodeSearchMode) String() string {
	if mode == CodeSearchModeUnion {
		return "union"
	}
	return "exact"
}

type SearchOptions struct {
	RepoIDs  []int64
	Keyword  string
	Language string
	Filename string

	Mode CodeSearchMode

	db.Paginator
}

// NewDummyIndexer returns a dummy indexer
func NewDummyIndexer() Indexer {
	return &dummyIndexer{
		Indexer: internal.NewDummyIndexer(),
	}
}

type dummyIndexer struct {
	internal.Indexer
}

func (d *dummyIndexer) Index(ctx context.Context, repo *repo_model.Repository, sha string, changes *RepoChanges) error {
	return errors.New("indexer is not ready")
}

func (d *dummyIndexer) Delete(ctx context.Context, repoID int64) error {
	return errors.New("indexer is not ready")
}

func (d *dummyIndexer) Search(ctx context.Context, opts *SearchOptions) (int64, []*SearchResult, []*SearchResultLanguages, error) {
	return 0, nil, nil, errors.New("indexer is not ready")
}
