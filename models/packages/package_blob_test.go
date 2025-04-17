// Copyright 2025 The Forgejo Authors.
// SPDX-License-Identifier: GPL-3.0-or-later

package packages

import (
	"testing"

	"forgejo.org/models/unittest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPackagesGetOrInsertBlob(t *testing.T) {
	defer AddFixtures("models/fixtures/TestPackagesGetOrInsertBlob/")()
	require.NoError(t, unittest.PrepareTestDatabase())

	blake2bIsSet := unittest.AssertExistsAndLoadBean(t, &PackageBlob{ID: 1})
	blake2bNotSet := unittest.AssertExistsAndLoadBean(t, &PackageBlob{ID: 2})

	var blake2bSetToRandom PackageBlob
	blake2bSetToRandom = *blake2bNotSet
	blake2bSetToRandom.HashBlake2b = "SOMETHING RANDOM"

	for _, testCase := range []struct {
		name        string
		exists      bool
		packageBlob *PackageBlob
	}{
		{
			name:        "exists and blake2b is not null in the database",
			exists:      true,
			packageBlob: blake2bIsSet,
		},
		{
			name:        "exists and blake2b is null in the database",
			exists:      true,
			packageBlob: &blake2bSetToRandom,
		},
		{
			name:   "does not exists",
			exists: false,
			packageBlob: &PackageBlob{
				Size:        30,
				HashMD5:     "HASHMD5_3",
				HashSHA1:    "HASHSHA1_3",
				HashSHA256:  "HASHSHA256_3",
				HashSHA512:  "HASHSHA512_3",
				HashBlake2b: "HASHBLAKE2B_3",
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			found, has, _ := GetOrInsertBlob(t.Context(), testCase.packageBlob)
			assert.Equal(t, testCase.exists, has)
			require.NotNil(t, found)
			if testCase.exists {
				assert.Equal(t, found.ID, testCase.packageBlob.ID)
			} else {
				unittest.BeanExists(t, &PackageBlob{ID: found.ID})
			}
		})
	}
}
