// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"net/http"
	"testing"

	"forgejo.org/modules/setting"
	"forgejo.org/modules/test"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

func TestAPISSHSigningKey(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	t.Run("No signing key", func(t *testing.T) {
		defer test.MockVariableValue(&setting.SSHInstanceKey, nil)()
		defer tests.PrintCurrentTest(t)()

		MakeRequest(t, NewRequest(t, "GET", "/api/v1/signing-key.ssh"), http.StatusNotFound)
	})
	t.Run("With signing key", func(t *testing.T) {
		publicKey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFeRC8GfFyXtiy0f1E7hLv77BXW7e68tFvIcs8/29YqH\n"
		pubKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(publicKey))
		require.NoError(t, err)
		defer test.MockVariableValue(&setting.SSHInstanceKey, pubKey)()
		defer tests.PrintCurrentTest(t)()

		resp := MakeRequest(t, NewRequest(t, "GET", "/api/v1/signing-key.ssh"), http.StatusOK)
		assert.Equal(t, publicKey, resp.Body.String())
	})
}
