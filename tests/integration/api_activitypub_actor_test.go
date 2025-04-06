// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"testing"

	"forgejo.org/modules/setting"
	"forgejo.org/modules/test"
	"forgejo.org/routers"
	"forgejo.org/tests"

	ap "github.com/go-ap/activitypub"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActivityPubActor(t *testing.T) {
	defer test.MockVariableValue(&setting.Federation.Enabled, true)()
	defer test.MockVariableValue(&testWebRoutes, routers.NormalRoutes())()
	defer tests.PrepareTestEnv(t)()

	req := NewRequest(t, "GET", "/api/v1/activitypub/actor")
	resp := MakeRequest(t, req, http.StatusOK)
	assert.Contains(t, resp.Body.String(), "@context")

	var actor ap.Actor
	err := actor.UnmarshalJSON(resp.Body.Bytes())
	require.NoError(t, err)

	assert.Equal(t, ap.ApplicationType, actor.Type)
	assert.Equal(t, setting.Domain, actor.PreferredUsername.String())
	keyID := actor.GetID().String()
	assert.Regexp(t, "activitypub/actor$", keyID)
	assert.Regexp(t, "activitypub/actor/outbox$", actor.Outbox.GetID().String())
	assert.Regexp(t, "activitypub/actor/inbox$", actor.Inbox.GetID().String())

	pubKey := actor.PublicKey
	assert.NotNil(t, pubKey)
	publicKeyID := keyID + "#main-key"
	assert.Equal(t, pubKey.ID.String(), publicKeyID)

	pubKeyPem := pubKey.PublicKeyPem
	assert.NotNil(t, pubKeyPem)
	assert.Regexp(t, "^-----BEGIN PUBLIC KEY-----", pubKeyPem)
}
