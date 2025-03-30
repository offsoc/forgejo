// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repository

import (
	"bytes"
	"image"
	"image/png"
	"testing"

	"forgejo.org/models/db"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	"forgejo.org/modules/avatar"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUploadAvatar(t *testing.T) {
	// Generate image
	myImage := image.NewRGBA(image.Rect(0, 0, 1, 1))
	var buff bytes.Buffer
	png.Encode(&buff, myImage)

	require.NoError(t, unittest.PrepareTestDatabase())
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 10})

	err := UploadAvatar(db.DefaultContext, repo, buff.Bytes())
	require.NoError(t, err)
	assert.Equal(t, avatar.HashAvatar(repo.ID, buff.Bytes()), repo.Avatar)
}

func TestUploadBigAvatar(t *testing.T) {
	// Generate BIG image
	myImage := image.NewRGBA(image.Rect(0, 0, 5000, 1))
	var buff bytes.Buffer
	png.Encode(&buff, myImage)

	require.NoError(t, unittest.PrepareTestDatabase())
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 10})

	err := UploadAvatar(db.DefaultContext, repo, buff.Bytes())
	require.Error(t, err)
}

func TestDeleteAvatar(t *testing.T) {
	// Generate image
	myImage := image.NewRGBA(image.Rect(0, 0, 1, 1))
	var buff bytes.Buffer
	png.Encode(&buff, myImage)

	require.NoError(t, unittest.PrepareTestDatabase())
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 10})

	err := UploadAvatar(db.DefaultContext, repo, buff.Bytes())
	require.NoError(t, err)

	err = DeleteAvatar(db.DefaultContext, repo)
	require.NoError(t, err)

	assert.Empty(t, repo.Avatar)
}

func TestTemplateGenerateAvatar(t *testing.T) {
	// Generate image
	myImage := image.NewRGBA(image.Rect(0, 0, 1, 1))
	var buff bytes.Buffer
	png.Encode(&buff, myImage)

	require.NoError(t, unittest.PrepareTestDatabase())
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 10})

	// Upload Avatar
	err := UploadAvatar(db.DefaultContext, repo, buff.Bytes())
	require.NoError(t, err)
	assert.Equal(t, avatar.HashAvatar(repo.ID, buff.Bytes()), repo.Avatar)

	// Generate the Avatar for Another Repo
	genRepo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 11})
	err = generateAvatar(db.DefaultContext, repo, genRepo)
	require.NoError(t, err)
	assert.Equal(t, avatar.HashAvatar(genRepo.ID, buff.Bytes()), genRepo.Avatar)

	// Make sure The 2 Hashes are not the same
	assert.NotEqual(t, repo.Avatar, genRepo.Avatar)
}
