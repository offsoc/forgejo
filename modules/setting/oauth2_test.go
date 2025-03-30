// Copyright 2024 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package setting

import (
	"os"
	"testing"

	"forgejo.org/modules/generate"
	"forgejo.org/modules/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetGeneralSigningSecret(t *testing.T) {
	// when there is no general signing secret, it should be generated, and keep the same value
	generalSigningSecret.Store(nil)
	s1 := GetGeneralTokenSigningSecret()
	assert.NotNil(t, s1)
	s2 := GetGeneralTokenSigningSecret()
	assert.Equal(t, s1, s2)

	// the config value should always override any pre-generated value
	cfg, _ := NewConfigProviderFromData(`
[oauth2]
JWT_SECRET = BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB
`)
	defer test.MockVariableValue(&InstallLock, true)()
	loadOAuth2From(cfg)
	actual := GetGeneralTokenSigningSecret()
	expected, _ := generate.DecodeJwtSecret("BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB")
	assert.Len(t, actual, 32)
	assert.Equal(t, expected, actual)
}

func TestGetGeneralSigningSecretSave(t *testing.T) {
	defer test.MockVariableValue(&InstallLock, true)()

	old := GetGeneralTokenSigningSecret()
	assert.Len(t, old, 32)

	tmpFile := t.TempDir() + "/app.ini"
	_ = os.WriteFile(tmpFile, nil, 0o644)
	cfg, _ := NewConfigProviderFromFile(tmpFile)
	loadOAuth2From(cfg)
	generated := GetGeneralTokenSigningSecret()
	assert.Len(t, generated, 32)
	assert.NotEqual(t, old, generated)

	generalSigningSecret.Store(nil)
	cfg, _ = NewConfigProviderFromFile(tmpFile)
	loadOAuth2From(cfg)
	again := GetGeneralTokenSigningSecret()
	assert.Equal(t, generated, again)

	iniContent, err := os.ReadFile(tmpFile)
	require.NoError(t, err)
	assert.Contains(t, string(iniContent), "JWT_SECRET = ")
}
