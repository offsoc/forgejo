// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package gist_test

import (
	"testing"

	"code.gitea.io/gitea/models/db"
	gist_model "code.gitea.io/gitea/models/gist"
	"code.gitea.io/gitea/models/unittest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGistListLoadOwner(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	gistList := make(gist_model.GistList, 4)
	gistList[0] = unittest.AssertExistsAndLoadBean(t, &gist_model.Gist{ID: 1})
	gistList[1] = unittest.AssertExistsAndLoadBean(t, &gist_model.Gist{ID: 2})
	gistList[2] = unittest.AssertExistsAndLoadBean(t, &gist_model.Gist{ID: 3})
	gistList[3] = unittest.AssertExistsAndLoadBean(t, &gist_model.Gist{ID: 4})

	for _, gist := range gistList {
		assert.Nil(t, gist.Owner)
	}

	require.NoError(t, gistList.LoadOwner(db.DefaultContext))

	assert.Equal(t, int64(2), gistList[0].Owner.ID)
	assert.Equal(t, int64(2), gistList[1].Owner.ID)
	assert.Equal(t, int64(2), gistList[2].Owner.ID)
	assert.Equal(t, int64(3), gistList[3].Owner.ID)
}
