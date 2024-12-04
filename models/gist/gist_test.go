// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package gist_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"code.gitea.io/gitea/models/db"
	gist_model "code.gitea.io/gitea/models/gist"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/setting"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGistVisibilityFromName(t *testing.T) {
	visibility, err := gist_model.GistVisibilityFromName("public")
	require.NoError(t, err)
	assert.Equal(t, gist_model.GistVisibilityPublic, visibility)

	visibility, err = gist_model.GistVisibilityFromName("hidden")
	require.NoError(t, err)
	assert.Equal(t, gist_model.GistVisibilityHidden, visibility)

	visibility, err = gist_model.GistVisibilityFromName("private")
	require.NoError(t, err)
	assert.Equal(t, gist_model.GistVisibilityPrivate, visibility)

	_, err = gist_model.GistVisibilityFromName("invalid")
	assert.Error(t, err)
}

func TestGetGistByUUID(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	gist, err := gist_model.GetGistByUUID(db.DefaultContext, "df852aec")
	require.NoError(t, err)
	assert.Equal(t, int64(1), gist.ID)

	gist, err = gist_model.GetGistByUUID(db.DefaultContext, "invalid")
	assert.Error(t, err)
	assert.True(t, gist_model.IsErrGistNotExist(err))
	assert.Nil(t, gist)
}

func TestGistGetRepoPath(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	gist := unittest.AssertExistsAndLoadBean(t, &gist_model.Gist{ID: 1})

	assert.Equal(t, filepath.Join(setting.Gist.RootPath, "df852aec.git"), gist.GetRepoPath())
}

func TestGistLink(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	gist := unittest.AssertExistsAndLoadBean(t, &gist_model.Gist{ID: 1})

	assert.Equal(t, "/gists/df852aec", gist.Link())
}

func TestGistHTMLURL(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	gist := unittest.AssertExistsAndLoadBean(t, &gist_model.Gist{ID: 1})

	assert.Equal(t, fmt.Sprintf("%sgists/df852aec", setting.AppURL), gist.HTMLURL())
}

func TestGistLoadOwner(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	gist := unittest.AssertExistsAndLoadBean(t, &gist_model.Gist{ID: 1})

	assert.Nil(t, gist.Owner)

	require.NoError(t, gist.LoadOwner(db.DefaultContext))

	assert.Equal(t, int64(2), gist.Owner.ID)
}

func TestGistIsOwner(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	user3 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 3})
	gist := unittest.AssertExistsAndLoadBean(t, &gist_model.Gist{ID: 1})

	assert.False(t, gist.IsOwner(nil))
	assert.True(t, gist.IsOwner(user2))
	assert.False(t, gist.IsOwner(user3))
}

func TestGistHasAccess(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	publicGist := unittest.AssertExistsAndLoadBean(t, &gist_model.Gist{ID: 1})
	hiddenGist := unittest.AssertExistsAndLoadBean(t, &gist_model.Gist{ID: 2})
	privateGist := unittest.AssertExistsAndLoadBean(t, &gist_model.Gist{ID: 3})

	assert.True(t, publicGist.HasAccess(nil))
	assert.True(t, publicGist.HasAccess(user))

	assert.True(t, hiddenGist.HasAccess(nil))
	assert.True(t, hiddenGist.HasAccess(user))

	assert.False(t, privateGist.HasAccess(nil))
	assert.True(t, privateGist.HasAccess(user))
}
