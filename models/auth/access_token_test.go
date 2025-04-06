// Copyright 2016 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package auth_test

import (
	"testing"

	auth_model "forgejo.org/models/auth"
	"forgejo.org/models/db"
	"forgejo.org/models/unittest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAccessToken(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	token := &auth_model.AccessToken{
		UID:  3,
		Name: "Token C",
	}
	require.NoError(t, auth_model.NewAccessToken(db.DefaultContext, token))
	unittest.AssertExistsAndLoadBean(t, token)

	invalidToken := &auth_model.AccessToken{
		ID:   token.ID, // duplicate
		UID:  2,
		Name: "Token F",
	}
	require.Error(t, auth_model.NewAccessToken(db.DefaultContext, invalidToken))
}

func TestAccessTokenByNameExists(t *testing.T) {
	name := "Token Gitea"

	require.NoError(t, unittest.PrepareTestDatabase())
	token := &auth_model.AccessToken{
		UID:  3,
		Name: name,
	}

	// Check to make sure it doesn't exists already
	exist, err := auth_model.AccessTokenByNameExists(db.DefaultContext, token)
	require.NoError(t, err)
	assert.False(t, exist)

	// Save it to the database
	require.NoError(t, auth_model.NewAccessToken(db.DefaultContext, token))
	unittest.AssertExistsAndLoadBean(t, token)

	// This token must be found by name in the DB now
	exist, err = auth_model.AccessTokenByNameExists(db.DefaultContext, token)
	require.NoError(t, err)
	assert.True(t, exist)

	user4Token := &auth_model.AccessToken{
		UID:  4,
		Name: name,
	}

	// Name matches but different user ID, this shouldn't exists in the
	// database
	exist, err = auth_model.AccessTokenByNameExists(db.DefaultContext, user4Token)
	require.NoError(t, err)
	assert.False(t, exist)
}

func TestGetAccessTokenBySHA(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	token, err := auth_model.GetAccessTokenBySHA(db.DefaultContext, "d2c6c1ba3890b309189a8e618c72a162e4efbf36")
	require.NoError(t, err)
	assert.Equal(t, int64(1), token.UID)
	assert.Equal(t, "Token A", token.Name)
	assert.Equal(t, "2b3668e11cb82d3af8c6e4524fc7841297668f5008d1626f0ad3417e9fa39af84c268248b78c481daa7e5dc437784003494f", token.TokenHash)
	assert.Equal(t, "e4efbf36", token.TokenLastEight)

	_, err = auth_model.GetAccessTokenBySHA(db.DefaultContext, "notahash")
	require.Error(t, err)
	assert.True(t, auth_model.IsErrAccessTokenNotExist(err))

	_, err = auth_model.GetAccessTokenBySHA(db.DefaultContext, "")
	require.Error(t, err)
	assert.True(t, auth_model.IsErrAccessTokenEmpty(err))
}

func TestListAccessTokens(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	tokens, err := db.Find[auth_model.AccessToken](db.DefaultContext, auth_model.ListAccessTokensOptions{UserID: 1})
	require.NoError(t, err)
	if assert.Len(t, tokens, 2) {
		assert.Equal(t, int64(1), tokens[0].UID)
		assert.Equal(t, int64(1), tokens[1].UID)
		assert.Contains(t, []string{tokens[0].Name, tokens[1].Name}, "Token A")
		assert.Contains(t, []string{tokens[0].Name, tokens[1].Name}, "Token B")
	}

	tokens, err = db.Find[auth_model.AccessToken](db.DefaultContext, auth_model.ListAccessTokensOptions{UserID: 2})
	require.NoError(t, err)
	if assert.Len(t, tokens, 1) {
		assert.Equal(t, int64(2), tokens[0].UID)
		assert.Equal(t, "Token A", tokens[0].Name)
	}

	tokens, err = db.Find[auth_model.AccessToken](db.DefaultContext, auth_model.ListAccessTokensOptions{UserID: 100})
	require.NoError(t, err)
	assert.Empty(t, tokens)
}

func TestUpdateAccessToken(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	token, err := auth_model.GetAccessTokenBySHA(db.DefaultContext, "4c6f36e6cf498e2a448662f915d932c09c5a146c")
	require.NoError(t, err)
	token.Name = "Token Z"

	require.NoError(t, auth_model.UpdateAccessToken(db.DefaultContext, token))
	unittest.AssertExistsAndLoadBean(t, token)
}

func TestDeleteAccessTokenByID(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	token, err := auth_model.GetAccessTokenBySHA(db.DefaultContext, "4c6f36e6cf498e2a448662f915d932c09c5a146c")
	require.NoError(t, err)
	assert.Equal(t, int64(1), token.UID)

	require.NoError(t, auth_model.DeleteAccessTokenByID(db.DefaultContext, token.ID, 1))
	unittest.AssertNotExistsBean(t, token)

	err = auth_model.DeleteAccessTokenByID(db.DefaultContext, 100, 100)
	require.Error(t, err)
	assert.True(t, auth_model.IsErrAccessTokenNotExist(err))
}

func TestRegenerateAccessTokenByID(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	token, err := auth_model.GetAccessTokenBySHA(db.DefaultContext, "4c6f36e6cf498e2a448662f915d932c09c5a146c")
	require.NoError(t, err)

	newToken, err := auth_model.RegenerateAccessTokenByID(db.DefaultContext, token.ID, 1)
	require.NoError(t, err)
	unittest.AssertNotExistsBean(t, &auth_model.AccessToken{ID: token.ID, UID: token.UID, TokenHash: token.TokenHash})
	newToken = &auth_model.AccessToken{
		ID:        newToken.ID,
		UID:       newToken.UID,
		TokenHash: newToken.TokenHash,
	}
	unittest.AssertExistsAndLoadBean(t, newToken)

	// Token has been recreated, new salt and hash, but should retain the same ID, UID, Name and Scope
	assert.Equal(t, token.ID, newToken.ID)
	assert.NotEqual(t, token.TokenHash, newToken.TokenHash)
	assert.NotEqual(t, token.TokenSalt, newToken.TokenSalt)
	assert.Equal(t, token.UID, newToken.UID)
	assert.Equal(t, token.Name, newToken.Name)
	assert.Equal(t, token.Scope, newToken.Scope)
}
