// Copyright 2024 The Forgejo Authors
// SPDX-License-Identifier: MIT

package activitypub

import (
	"testing"

	"code.gitea.io/gitea/models/db"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/unittest"

	_ "code.gitea.io/gitea/models" // https://forum.gitea.com/t/testfixtures-could-not-clean-table-access-no-such-table-access/4137/4

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepoKeyPair(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	repo1 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	pub, priv, err := GetRepoKeyPair(db.DefaultContext, repo1)
	require.NoError(t, err)
	pub1, err := GetRepoPublicKey(db.DefaultContext, repo1)
	require.NoError(t, err)
	assert.Equal(t, pub, pub1)
	priv1, err := GetRepoPrivateKey(db.DefaultContext, repo1)
	require.NoError(t, err)
	assert.Equal(t, priv, priv1)
}
