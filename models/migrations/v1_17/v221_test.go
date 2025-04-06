// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package v1_17 //nolint

import (
	"encoding/base32"
	"testing"

	migration_tests "forgejo.org/models/migrations/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_StoreWebauthnCredentialIDAsBytes(t *testing.T) {
	// Create webauthnCredential table
	type WebauthnCredential struct {
		ID              int64 `xorm:"pk autoincr"`
		Name            string
		LowerName       string `xorm:"unique(s)"`
		UserID          int64  `xorm:"INDEX unique(s)"`
		CredentialID    string `xorm:"INDEX VARCHAR(410)"`
		PublicKey       []byte
		AttestationType string
		AAGUID          []byte
		SignCount       uint32 `xorm:"BIGINT"`
		CloneWarning    bool
	}

	type ExpectedWebauthnCredential struct {
		ID           int64  `xorm:"pk autoincr"`
		CredentialID string // CredentialID is at most 1023 bytes as per spec released 20 July 2022
	}

	type ConvertedWebauthnCredential struct {
		ID                int64  `xorm:"pk autoincr"`
		CredentialIDBytes []byte `xorm:"VARBINARY(1024)"` // CredentialID is at most 1023 bytes as per spec released 20 July 2022
	}

	// Prepare and load the testing database
	x, deferable := migration_tests.PrepareTestEnv(t, 0, new(WebauthnCredential), new(ExpectedWebauthnCredential))
	defer deferable()
	if x == nil || t.Failed() {
		return
	}

	err := StoreWebauthnCredentialIDAsBytes(x)
	require.NoError(t, err)

	expected := []ExpectedWebauthnCredential{}
	err = x.Table("expected_webauthn_credential").Asc("id").Find(&expected)
	require.NoError(t, err)

	got := []ConvertedWebauthnCredential{}
	err = x.Table("webauthn_credential").Select("id, credential_id_bytes").Asc("id").Find(&got)
	require.NoError(t, err)

	for i, e := range expected {
		credIDBytes, _ := base32.HexEncoding.DecodeString(e.CredentialID)
		assert.Equal(t, credIDBytes, got[i].CredentialIDBytes)
	}
}
