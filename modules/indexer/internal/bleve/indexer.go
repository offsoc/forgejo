// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package bleve

import (
	"context"
	"errors"

	"forgejo.org/modules/indexer/internal"
	"forgejo.org/modules/log"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/mapping"
)

var _ internal.Indexer = &Indexer{}

// Indexer represents a basic bleve indexer implementation
type Indexer struct {
	Indexer bleve.Index

	indexDir      string
	version       int
	mappingGetter MappingGetter
}

type MappingGetter func() (mapping.IndexMapping, error)

func NewIndexer(indexDir string, version int, mappingGetter func() (mapping.IndexMapping, error)) *Indexer {
	return &Indexer{
		indexDir:      indexDir,
		version:       version,
		mappingGetter: mappingGetter,
	}
}

// Init initializes the indexer
func (i *Indexer) Init(_ context.Context) (bool, error) {
	if i == nil {
		return false, errors.New("cannot init nil indexer")
	}

	if i.Indexer != nil {
		return false, errors.New("indexer is already initialized")
	}

	indexer, version, err := openIndexer(i.indexDir, i.version)
	if err != nil {
		return false, err
	}
	if indexer != nil {
		i.Indexer = indexer
		return true, nil
	}

	if version != 0 {
		log.Warn("Found older bleve index with version %d, Forgejo will remove it and rebuild", version)
	}

	indexMapping, err := i.mappingGetter()
	if err != nil {
		return false, err
	}

	indexer, err = bleve.New(i.indexDir, indexMapping)
	if err != nil {
		return false, err
	}

	if err = writeIndexMetadata(i.indexDir, &IndexMetadata{
		Version: i.version,
	}); err != nil {
		return false, err
	}

	i.Indexer = indexer

	return false, nil
}

// Ping checks if the indexer is available
func (i *Indexer) Ping(_ context.Context) error {
	if i == nil {
		return errors.New("cannot ping nil indexer")
	}
	if i.Indexer == nil {
		return errors.New("indexer is not initialized")
	}
	return nil
}

func (i *Indexer) Close() {
	if i == nil || i.Indexer == nil {
		return
	}

	if err := i.Indexer.Close(); err != nil {
		log.Error("Failed to close bleve indexer in %q: %v", i.indexDir, err)
	}
	i.Indexer = nil
}
