// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package webauthn

import (
	"testing"

	"forgejo.org/modules/setting"
	"forgejo.org/modules/test"

	"github.com/stretchr/testify/assert"
)

func TestInit(t *testing.T) {
	defer test.MockVariableValue(&setting.Domain, "domain")()
	defer test.MockVariableValue(&setting.AppName, "AppName")()
	defer test.MockVariableValue(&setting.AppURL, "https://domain/")()

	Init()

	assert.Equal(t, setting.Domain, WebAuthn.Config.RPID)
	assert.Equal(t, setting.AppName, WebAuthn.Config.RPDisplayName)
	assert.Equal(t, []string{"https://domain"}, WebAuthn.Config.RPOrigins)
}
