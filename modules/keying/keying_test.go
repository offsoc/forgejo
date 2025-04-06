// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package keying_test

import (
	"math"
	"testing"

	"forgejo.org/modules/keying"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/chacha20poly1305"
)

func TestKeying(t *testing.T) {
	t.Run("Not initialized", func(t *testing.T) {
		assert.Panics(t, func() {
			keying.DeriveKey(keying.Context("TESTING"))
		})
	})

	t.Run("Initialization", func(t *testing.T) {
		keying.Init([]byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07})
	})

	t.Run("Context separation", func(t *testing.T) {
		key1 := keying.DeriveKey(keying.Context("TESTING"))
		key2 := keying.DeriveKey(keying.Context("TESTING2"))

		ciphertext := key1.Encrypt([]byte("This is for context TESTING"), nil)

		plaintext, err := key2.Decrypt(ciphertext, nil)
		require.Error(t, err)
		assert.Empty(t, plaintext)

		plaintext, err = key1.Decrypt(ciphertext, nil)
		require.NoError(t, err)
		assert.EqualValues(t, "This is for context TESTING", plaintext)
	})

	context := keying.Context("TESTING PURPOSES")
	plainText := []byte("Forgejo is run by [Redacted]")
	var cipherText []byte
	t.Run("Encrypt", func(t *testing.T) {
		key := keying.DeriveKey(context)

		cipherText = key.Encrypt(plainText, []byte{0x05, 0x06})
		cipherText2 := key.Encrypt(plainText, []byte{0x05, 0x06})

		// Ensure ciphertexts don't have an deterministic output.
		assert.NotEqual(t, cipherText, cipherText2)
	})

	t.Run("Decrypt", func(t *testing.T) {
		key := keying.DeriveKey(context)

		t.Run("Successful", func(t *testing.T) {
			convertedPlainText, err := key.Decrypt(cipherText, []byte{0x05, 0x06})
			require.NoError(t, err)
			assert.Equal(t, plainText, convertedPlainText)
		})

		t.Run("Not enough additional data", func(t *testing.T) {
			plainText, err := key.Decrypt(cipherText, []byte{0x05})
			require.Error(t, err)
			assert.Empty(t, plainText)
		})

		t.Run("Too much additional data", func(t *testing.T) {
			plainText, err := key.Decrypt(cipherText, []byte{0x05, 0x06, 0x07})
			require.Error(t, err)
			assert.Empty(t, plainText)
		})

		t.Run("Incorrect nonce", func(t *testing.T) {
			// Flip the first byte of the nonce.
			cipherText[0] = ^cipherText[0]

			plainText, err := key.Decrypt(cipherText, []byte{0x05, 0x06})
			require.Error(t, err)
			assert.Empty(t, plainText)
		})

		t.Run("Incorrect ciphertext", func(t *testing.T) {
			assert.Panics(t, func() {
				key.Decrypt(nil, nil)
			})

			assert.Panics(t, func() {
				cipherText := make([]byte, chacha20poly1305.NonceSizeX)
				key.Decrypt(cipherText, nil)
			})
		})
	})
}

func TestKeyingColumnAndID(t *testing.T) {
	assert.Equal(t, []byte{0x74, 0x61, 0x62, 0x6c, 0x65, 0x3a, 0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, keying.ColumnAndID("table", math.MinInt64))
	assert.Equal(t, []byte{0x74, 0x61, 0x62, 0x6c, 0x65, 0x3a, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}, keying.ColumnAndID("table", -1))
	assert.Equal(t, []byte{0x74, 0x61, 0x62, 0x6c, 0x65, 0x3a, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, keying.ColumnAndID("table", 0))
	assert.Equal(t, []byte{0x74, 0x61, 0x62, 0x6c, 0x65, 0x3a, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}, keying.ColumnAndID("table", 1))
	assert.Equal(t, []byte{0x74, 0x61, 0x62, 0x6c, 0x65, 0x3a, 0x7f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}, keying.ColumnAndID("table", math.MaxInt64))

	assert.Equal(t, []byte{0x74, 0x61, 0x62, 0x6c, 0x65, 0x32, 0x3a, 0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, keying.ColumnAndID("table2", math.MinInt64))
	assert.Equal(t, []byte{0x74, 0x61, 0x62, 0x6c, 0x65, 0x32, 0x3a, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}, keying.ColumnAndID("table2", -1))
	assert.Equal(t, []byte{0x74, 0x61, 0x62, 0x6c, 0x65, 0x32, 0x3a, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, keying.ColumnAndID("table2", 0))
	assert.Equal(t, []byte{0x74, 0x61, 0x62, 0x6c, 0x65, 0x32, 0x3a, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}, keying.ColumnAndID("table2", 1))
	assert.Equal(t, []byte{0x74, 0x61, 0x62, 0x6c, 0x65, 0x32, 0x3a, 0x7f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}, keying.ColumnAndID("table2", math.MaxInt64))
}
