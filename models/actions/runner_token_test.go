// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package actions

import (
	"testing"

	"forgejo.org/models/db"
	"forgejo.org/models/unittest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetLatestRunnerToken(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	token := unittest.AssertExistsAndLoadBean(t, &ActionRunnerToken{ID: 3})
	expectedToken, err := GetLatestRunnerToken(db.DefaultContext, 1, 0)
	require.NoError(t, err)
	assert.Equal(t, expectedToken, token)
}

func TestNewRunnerToken(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	token, err := NewRunnerToken(db.DefaultContext, 1, 0)
	require.NoError(t, err)
	expectedToken, err := GetLatestRunnerToken(db.DefaultContext, 1, 0)
	require.NoError(t, err)
	assert.Equal(t, expectedToken, token)
}

func TestUpdateRunnerToken(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	token := unittest.AssertExistsAndLoadBean(t, &ActionRunnerToken{ID: 3})
	token.IsActive = true
	require.NoError(t, UpdateRunnerToken(db.DefaultContext, token))
	expectedToken, err := GetLatestRunnerToken(db.DefaultContext, 1, 0)
	require.NoError(t, err)
	assert.Equal(t, expectedToken, token)
}
