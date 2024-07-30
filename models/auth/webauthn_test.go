// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package auth_test

import (
	"testing"

	auth_model "code.gitea.io/gitea/models/auth"
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/unittest"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetWebAuthnCredentialByID(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	res, err := auth_model.GetWebAuthnCredentialByID(db.DefaultContext, 1)
	require.NoError(t, err)
	assert.Equal(t, "WebAuthn credential", res.Name)

	_, err = auth_model.GetWebAuthnCredentialByID(db.DefaultContext, 342432)
	require.Error(t, err)
	assert.True(t, auth_model.IsErrWebAuthnCredentialNotExist(err))
}

func TestGetWebAuthnCredentialsByUID(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	res, err := auth_model.GetWebAuthnCredentialsByUID(db.DefaultContext, 32)
	require.NoError(t, err)
	assert.Len(t, res, 1)
	assert.Equal(t, "WebAuthn credential", res[0].Name)
}

func TestWebAuthnCredential_TableName(t *testing.T) {
	assert.Equal(t, "webauthn_credential", auth_model.WebAuthnCredential{}.TableName())
}

func TestWebAuthnCredential_UpdateSignCount(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	cred := unittest.AssertExistsAndLoadBean(t, &auth_model.WebAuthnCredential{ID: 1})
	cred.SignCount = 1
	require.NoError(t, cred.UpdateSignCount(db.DefaultContext))
	unittest.AssertExistsIf(t, true, &auth_model.WebAuthnCredential{ID: 1, SignCount: 1})
}

func TestWebAuthnCredential_UpdateLargeCounter(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	cred := unittest.AssertExistsAndLoadBean(t, &auth_model.WebAuthnCredential{ID: 1})
	cred.SignCount = 0xffffffff
	require.NoError(t, cred.UpdateSignCount(db.DefaultContext))
	unittest.AssertExistsIf(t, true, &auth_model.WebAuthnCredential{ID: 1, SignCount: 0xffffffff})
}

func TestCreateCredential(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	res, err := auth_model.CreateCredential(db.DefaultContext, 1, "WebAuthn Created Credential", &webauthn.Credential{ID: []byte("Test")})
	require.NoError(t, err)
	assert.Equal(t, "WebAuthn Created Credential", res.Name)
	assert.Equal(t, []byte("Test"), res.CredentialID)

	unittest.AssertExistsIf(t, true, &auth_model.WebAuthnCredential{Name: "WebAuthn Created Credential", UserID: 1})
}
