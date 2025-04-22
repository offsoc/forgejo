// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package setting

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

func TestSSHInstanceKey(t *testing.T) {
	sshSigningKeyPath, err := filepath.Abs("../../tests/integration/ssh-signing-key.pub")
	require.NoError(t, err)

	t.Run("None value", func(t *testing.T) {
		cfg, err := NewConfigProviderFromData(`
[repository.signing]
FORMAT = ssh
SIGNING_KEY = none
`)
		require.NoError(t, err)

		loadRepositoryFrom(cfg)

		assert.Nil(t, SSHInstanceKey)
	})

	t.Run("No value", func(t *testing.T) {
		cfg, err := NewConfigProviderFromData(`
[repository.signing]
FORMAT = ssh
`)
		require.NoError(t, err)

		loadRepositoryFrom(cfg)

		assert.Nil(t, SSHInstanceKey)
	})
	t.Run("Normal", func(t *testing.T) {
		iniStr := fmt.Sprintf(`
[repository.signing]
FORMAT = ssh
SIGNING_KEY = %s
`, sshSigningKeyPath)
		cfg, err := NewConfigProviderFromData(iniStr)
		require.NoError(t, err)

		loadRepositoryFrom(cfg)

		assert.NotNil(t, SSHInstanceKey)
		assert.Equal(t, "ssh-ed25519", SSHInstanceKey.Type())
		assert.EqualValues(t, "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFeRC8GfFyXtiy0f1E7hLv77BXW7e68tFvIcs8/29YqH\n", ssh.MarshalAuthorizedKey(SSHInstanceKey))
	})
}
