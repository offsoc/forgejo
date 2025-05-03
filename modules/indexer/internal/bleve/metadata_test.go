// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

// Copied and modified from https://github.com/ethantkoenig/rupture (MIT License)

package bleve

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetadata(t *testing.T) {
	dir := t.TempDir()

	meta, err := readIndexMetadata(dir)
	require.NoError(t, err)
	assert.Equal(t, &IndexMetadata{}, meta)

	meta.Version = 24
	require.NoError(t, writeIndexMetadata(dir, meta))

	meta, err = readIndexMetadata(dir)
	require.NoError(t, err)
	assert.Equal(t, 24, meta.Version)
}
