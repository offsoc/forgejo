// Copyright 2020 The Gitea Authors. All rights reserved.
// Copyright 2025 The Forgejo Authors. All rights reserved
// SPDX-License-Identifier: MIT

package setting

import (
	"testing"

	"code.gitea.io/gitea/modules/json"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMakeAbsoluteAssetURL(t *testing.T) {
	assert.Equal(t, "https://localhost:2345", MakeAbsoluteAssetURL("https://localhost:1234", "https://localhost:2345"))
	assert.Equal(t, "https://localhost:2345", MakeAbsoluteAssetURL("https://localhost:1234/", "https://localhost:2345"))
	assert.Equal(t, "https://localhost:2345", MakeAbsoluteAssetURL("https://localhost:1234/", "https://localhost:2345/"))
	assert.Equal(t, "https://localhost:1234/foo", MakeAbsoluteAssetURL("https://localhost:1234", "/foo"))
	assert.Equal(t, "https://localhost:1234/foo", MakeAbsoluteAssetURL("https://localhost:1234/", "/foo"))
	assert.Equal(t, "https://localhost:1234/foo", MakeAbsoluteAssetURL("https://localhost:1234/", "/foo/"))
	assert.Equal(t, "https://localhost:1234/foo", MakeAbsoluteAssetURL("https://localhost:1234/foo", "/foo"))
	assert.Equal(t, "https://localhost:1234/foo", MakeAbsoluteAssetURL("https://localhost:1234/foo/", "/foo"))
	assert.Equal(t, "https://localhost:1234/foo", MakeAbsoluteAssetURL("https://localhost:1234/foo/", "/foo/"))
	assert.Equal(t, "https://localhost:1234/bar", MakeAbsoluteAssetURL("https://localhost:1234/foo", "/bar"))
	assert.Equal(t, "https://localhost:1234/bar", MakeAbsoluteAssetURL("https://localhost:1234/foo/", "/bar"))
	assert.Equal(t, "https://localhost:1234/bar", MakeAbsoluteAssetURL("https://localhost:1234/foo/", "/bar/"))
}

func TestMakeManifestData(t *testing.T) {
	jsonBytes := MakeManifestData(`Example App '\"`, "https://example.com", "https://example.com/foo/bar")
	assert.True(t, json.Valid(jsonBytes))
}

func TestLoadServiceDomainListsForFederation(t *testing.T) {
	oldAppURL := AppURL
	oldFederation := Federation
	oldService := Service

	defer func() {
		AppURL = oldAppURL
		Federation = oldFederation
		Service = oldService
	}()

	cfg, err := NewConfigProviderFromData(`
[federation]
ENABLED = true
[service]
EMAIL_DOMAIN_ALLOWLIST = *.allow.random
EMAIL_DOMAIN_BLOCKLIST = *.block.random
`)

	require.NoError(t, err)
	loadServerFrom(cfg)
	loadFederationFrom(cfg)
	loadServiceFrom(cfg)

	assert.True(t, match(Service.EmailDomainAllowList, "d1.allow.random"))
	assert.True(t, match(Service.EmailDomainAllowList, "localhost"))
}

func TestLoadServiceDomainListsNoFederation(t *testing.T) {
	oldAppURL := AppURL
	oldFederation := Federation
	oldService := Service

	defer func() {
		AppURL = oldAppURL
		Federation = oldFederation
		Service = oldService
	}()

	cfg, err := NewConfigProviderFromData(`
[federation]
ENABLED = false
[service]
EMAIL_DOMAIN_ALLOWLIST = *.allow.random
EMAIL_DOMAIN_BLOCKLIST = *.block.random
`)

	require.NoError(t, err)
	loadServerFrom(cfg)
	loadFederationFrom(cfg)
	loadServiceFrom(cfg)

	assert.True(t, match(Service.EmailDomainAllowList, "d1.allow.random"))
}

func TestLoadServiceDomainListsFederationEmptyAllowList(t *testing.T) {
	oldAppURL := AppURL
	oldFederation := Federation
	oldService := Service

	defer func() {
		AppURL = oldAppURL
		Federation = oldFederation
		Service = oldService
	}()

	cfg, err := NewConfigProviderFromData(`
[federation]
ENABLED = true
[service]
EMAIL_DOMAIN_BLOCKLIST = *.block.random
`)

	require.NoError(t, err)
	loadServerFrom(cfg)
	loadFederationFrom(cfg)
	loadServiceFrom(cfg)

	assert.Empty(t, Service.EmailDomainAllowList)
}
