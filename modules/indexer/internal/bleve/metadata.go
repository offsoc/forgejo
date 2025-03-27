// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

// Copied and modified from https://github.com/ethantkoenig/rupture (MIT License)

package bleve

import (
	"os"
	"path/filepath"

	"forgejo.org/modules/json"
)

const metaFilename = "rupture_meta.json"

func indexMetadataPath(dir string) string {
	return filepath.Join(dir, metaFilename)
}

// IndexMetadata contains metadata about a bleve index.
type IndexMetadata struct {
	// The version of the data in the index. This can be useful for tracking
	// schema changes or data migrations.
	Version int `json:"version"`
}

// readIndexMetadata returns the metadata for the index at the specified path.
// If no such index metadata exists, an empty metadata and a nil error are
// returned.
func readIndexMetadata(path string) (*IndexMetadata, error) {
	meta := &IndexMetadata{}
	metaPath := indexMetadataPath(path)
	if _, err := os.Stat(metaPath); os.IsNotExist(err) {
		return meta, nil
	} else if err != nil {
		return nil, err
	}

	metaBytes, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, err
	}
	return meta, json.Unmarshal(metaBytes, &meta)
}

// writeIndexMetadata writes metadata for the index at the specified path.
func writeIndexMetadata(path string, meta *IndexMetadata) error {
	metaBytes, err := json.Marshal(meta)
	if err != nil {
		return err
	}

	return os.WriteFile(indexMetadataPath(path), metaBytes, 0o644)
}
