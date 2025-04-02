// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package repo

import (
	"testing"

	"forgejo.org/models/unittest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReleaseListLoadAttributes(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	releases := ReleaseList{&Release{
		RepoID:      1,
		PublisherID: 1,
	}, &Release{
		RepoID:      2,
		PublisherID: 2,
	}, &Release{
		RepoID:      1,
		PublisherID: 2,
	}, &Release{
		RepoID:      2,
		PublisherID: 1,
	}}

	require.NoError(t, releases.LoadAttributes(t.Context()))

	assert.EqualValues(t, 1, releases[0].Repo.ID)
	assert.EqualValues(t, 1, releases[0].Publisher.ID)
	assert.EqualValues(t, 2, releases[1].Repo.ID)
	assert.EqualValues(t, 2, releases[1].Publisher.ID)
	assert.EqualValues(t, 1, releases[2].Repo.ID)
	assert.EqualValues(t, 2, releases[2].Publisher.ID)
	assert.EqualValues(t, 2, releases[3].Repo.ID)
	assert.EqualValues(t, 1, releases[3].Publisher.ID)
}
