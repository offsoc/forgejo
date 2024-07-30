// Copyright 2018 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package git

import (
	"fmt"
	"io"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepository_GetBlob_Found(t *testing.T) {
	repoPath := filepath.Join(testReposDir, "repo1_bare")
	r, err := openRepositoryWithDefaultContext(repoPath)
	require.NoError(t, err)
	defer r.Close()

	testCases := []struct {
		OID  string
		Data []byte
	}{
		{"e2129701f1a4d54dc44f03c93bca0a2aec7c5449", []byte("file1\n")},
		{"6c493ff740f9380390d5c9ddef4af18697ac9375", []byte("file2\n")},
	}

	for _, testCase := range testCases {
		blob, err := r.GetBlob(testCase.OID)
		require.NoError(t, err)

		dataReader, err := blob.DataAsync()
		require.NoError(t, err)

		data, err := io.ReadAll(dataReader)
		require.NoError(t, dataReader.Close())
		require.NoError(t, err)
		assert.Equal(t, testCase.Data, data)
	}
}

func TestRepository_GetBlob_NotExist(t *testing.T) {
	repoPath := filepath.Join(testReposDir, "repo1_bare")
	r, err := openRepositoryWithDefaultContext(repoPath)
	require.NoError(t, err)
	defer r.Close()

	testCase := "0000000000000000000000000000000000000000"
	testError := ErrNotExist{testCase, ""}

	blob, err := r.GetBlob(testCase)
	assert.Nil(t, blob)
	assert.EqualError(t, err, testError.Error())
}

func TestRepository_GetBlob_NoId(t *testing.T) {
	repoPath := filepath.Join(testReposDir, "repo1_bare")
	r, err := openRepositoryWithDefaultContext(repoPath)
	require.NoError(t, err)
	defer r.Close()

	testCase := ""
	testError := fmt.Errorf("length %d has no matched object format: %s", len(testCase), testCase)

	blob, err := r.GetBlob(testCase)
	assert.Nil(t, blob)
	assert.EqualError(t, err, testError.Error())
}
