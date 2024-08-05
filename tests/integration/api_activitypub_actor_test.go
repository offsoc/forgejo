// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"net/url"
	"testing"

	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/test"
	"code.gitea.io/gitea/routers"

	ap "github.com/go-ap/activitypub"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActivityPubActor(t *testing.T) {
	defer test.MockVariableValue(&setting.Federation.Enabled, true)()
	defer test.MockVariableValue(&testWebRoutes, routers.NormalRoutes())()

	onGiteaRun(t, func(*testing.T, *url.URL) {
		req := NewRequest(t, "GET", "/api/v1/activitypub/actor")
		resp := MakeRequest(t, req, http.StatusOK)
		body := resp.Body.Bytes()
		assert.Contains(t, string(body), "@context")

		var actor ap.Actor
		err := actor.UnmarshalJSON(body)
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
	})
}
