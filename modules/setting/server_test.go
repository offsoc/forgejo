// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package setting

import (
	"testing"

	"forgejo.org/modules/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDisplayNameDefault(t *testing.T) {
	defer test.MockVariableValue(&AppName, "Forgejo")()
	defer test.MockVariableValue(&AppSlogan, "Beyond coding. We Forge.")()
	defer test.MockVariableValue(&AppDisplayNameFormat, "{APP_NAME}: {APP_SLOGAN}")()
	displayName := generateDisplayName()
	assert.Equal(t, "Forgejo: Beyond coding. We Forge.", displayName)
}

func TestDisplayNameEmptySlogan(t *testing.T) {
	defer test.MockVariableValue(&AppName, "Forgejo")()
	defer test.MockVariableValue(&AppSlogan, "")()
	defer test.MockVariableValue(&AppDisplayNameFormat, "{APP_NAME}: {APP_SLOGAN}")()
	displayName := generateDisplayName()
	assert.Equal(t, "Forgejo", displayName)
}

func TestDisplayNameCustomFormat(t *testing.T) {
	defer test.MockVariableValue(&AppName, "Forgejo")()
	defer test.MockVariableValue(&AppSlogan, "Beyond coding. We Forge.")()
	defer test.MockVariableValue(&AppDisplayNameFormat, "{APP_NAME} - {APP_SLOGAN}")()
	displayName := generateDisplayName()
	assert.Equal(t, "Forgejo - Beyond coding. We Forge.", displayName)
}

func TestMaxUserRedirectsDefault(t *testing.T) {
	iniStr := ``
	cfg, err := NewConfigProviderFromData(iniStr)
	require.NoError(t, err)
	loadServiceFrom(cfg)

	assert.EqualValues(t, 0, Service.UsernameCooldownPeriod)
	assert.EqualValues(t, 0, Service.MaxUserRedirects)

	iniStr = `[service]
MAX_USER_REDIRECTS = 8`
	cfg, err = NewConfigProviderFromData(iniStr)
	require.NoError(t, err)
	loadServiceFrom(cfg)

	assert.EqualValues(t, 0, Service.UsernameCooldownPeriod)
	assert.EqualValues(t, 8, Service.MaxUserRedirects)

	iniStr = `[service]
USERNAME_COOLDOWN_PERIOD = 3`
	cfg, err = NewConfigProviderFromData(iniStr)
	require.NoError(t, err)
	loadServiceFrom(cfg)

	assert.EqualValues(t, 3, Service.UsernameCooldownPeriod)
	assert.EqualValues(t, 5, Service.MaxUserRedirects)

	iniStr = `[service]
USERNAME_COOLDOWN_PERIOD = 3
MAX_USER_REDIRECTS = 8`
	cfg, err = NewConfigProviderFromData(iniStr)
	require.NoError(t, err)
	loadServiceFrom(cfg)

	assert.EqualValues(t, 3, Service.UsernameCooldownPeriod)
	assert.EqualValues(t, 8, Service.MaxUserRedirects)
}

func TestUnixSocketAbstractNamespace(t *testing.T) {
	iniStr := `
	[server]
	PROTOCOL=http+unix
	HTTP_ADDR=@forgejo
	`
	cfg, err := NewConfigProviderFromData(iniStr)
	require.NoError(t, err)
	loadServerFrom(cfg)

	assert.Equal(t, "@forgejo", HTTPAddr)
}
