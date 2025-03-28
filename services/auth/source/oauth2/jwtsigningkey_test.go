// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package oauth2

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"forgejo.org/modules/setting"
	"forgejo.org/modules/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadOrCreateAsymmetricKey(t *testing.T) {
	loadKey := func(t *testing.T) any {
		t.Helper()
		loadOrCreateAsymmetricKey()

		fileContent, err := os.ReadFile(setting.OAuth2.JWTSigningPrivateKeyFile)
		require.NoError(t, err)

		block, _ := pem.Decode(fileContent)
		assert.NotNil(t, block)
		assert.Equal(t, "PRIVATE KEY", block.Type)

		parsedKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		require.NoError(t, err)

		return parsedKey
	}
	t.Run("RSA-2048", func(t *testing.T) {
		defer test.MockVariableValue(&setting.OAuth2.JWTSigningPrivateKeyFile, filepath.Join(t.TempDir(), "jwt-rsa-2048.priv"))()
		defer test.MockVariableValue(&setting.OAuth2.JWTSigningAlgorithm, "RS256")()

		parsedKey := loadKey(t)

		rsaPrivateKey := parsedKey.(*rsa.PrivateKey)
		assert.Equal(t, 2048, rsaPrivateKey.N.BitLen())

		t.Run("Load key with differ specified algorithm", func(t *testing.T) {
			defer test.MockVariableValue(&setting.OAuth2.JWTSigningAlgorithm, "EdDSA")()

			parsedKey := loadKey(t)
			rsaPrivateKey := parsedKey.(*rsa.PrivateKey)
			assert.Equal(t, 2048, rsaPrivateKey.N.BitLen())
		})
	})

	t.Run("RSA-3072", func(t *testing.T) {
		defer test.MockVariableValue(&setting.OAuth2.JWTSigningPrivateKeyFile, filepath.Join(t.TempDir(), "jwt-rsa-3072.priv"))()
		defer test.MockVariableValue(&setting.OAuth2.JWTSigningAlgorithm, "RS384")()

		parsedKey := loadKey(t)

		rsaPrivateKey := parsedKey.(*rsa.PrivateKey)
		assert.Equal(t, 3072, rsaPrivateKey.N.BitLen())
	})

	t.Run("RSA-4096", func(t *testing.T) {
		defer test.MockVariableValue(&setting.OAuth2.JWTSigningPrivateKeyFile, filepath.Join(t.TempDir(), "jwt-rsa-4096.priv"))()
		defer test.MockVariableValue(&setting.OAuth2.JWTSigningAlgorithm, "RS512")()

		parsedKey := loadKey(t)

		rsaPrivateKey := parsedKey.(*rsa.PrivateKey)
		assert.Equal(t, 4096, rsaPrivateKey.N.BitLen())
	})

	t.Run("ECDSA-256", func(t *testing.T) {
		defer test.MockVariableValue(&setting.OAuth2.JWTSigningPrivateKeyFile, filepath.Join(t.TempDir(), "jwt-ecdsa-256.priv"))()
		defer test.MockVariableValue(&setting.OAuth2.JWTSigningAlgorithm, "ES256")()

		parsedKey := loadKey(t)

		ecdsaPrivateKey := parsedKey.(*ecdsa.PrivateKey)
		assert.Equal(t, 256, ecdsaPrivateKey.Params().BitSize)
	})

	t.Run("ECDSA-384", func(t *testing.T) {
		defer test.MockVariableValue(&setting.OAuth2.JWTSigningPrivateKeyFile, filepath.Join(t.TempDir(), "jwt-ecdsa-384.priv"))()
		defer test.MockVariableValue(&setting.OAuth2.JWTSigningAlgorithm, "ES384")()

		parsedKey := loadKey(t)

		ecdsaPrivateKey := parsedKey.(*ecdsa.PrivateKey)
		assert.Equal(t, 384, ecdsaPrivateKey.Params().BitSize)
	})

	t.Run("ECDSA-512", func(t *testing.T) {
		defer test.MockVariableValue(&setting.OAuth2.JWTSigningPrivateKeyFile, filepath.Join(t.TempDir(), "jwt-ecdsa-512.priv"))()
		defer test.MockVariableValue(&setting.OAuth2.JWTSigningAlgorithm, "ES512")()

		parsedKey := loadKey(t)

		ecdsaPrivateKey := parsedKey.(*ecdsa.PrivateKey)
		assert.Equal(t, 521, ecdsaPrivateKey.Params().BitSize)
	})

	t.Run("EdDSA", func(t *testing.T) {
		defer test.MockVariableValue(&setting.OAuth2.JWTSigningPrivateKeyFile, filepath.Join(t.TempDir(), "jwt-eddsa.priv"))()
		defer test.MockVariableValue(&setting.OAuth2.JWTSigningAlgorithm, "EdDSA")()

		parsedKey := loadKey(t)

		assert.NotNil(t, parsedKey.(ed25519.PrivateKey))
	})
}
