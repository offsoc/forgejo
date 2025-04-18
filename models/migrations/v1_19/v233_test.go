// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package v1_19 //nolint

import (
	"testing"

	migration_tests "forgejo.org/models/migrations/test"
	"forgejo.org/modules/json"
	"forgejo.org/modules/secret"
	"forgejo.org/modules/setting"
	webhook_module "forgejo.org/modules/webhook"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_AddHeaderAuthorizationEncryptedColWebhook(t *testing.T) {
	// Create Webhook table
	type Webhook struct {
		ID   int64                   `xorm:"pk autoincr"`
		Type webhook_module.HookType `xorm:"VARCHAR(16) 'type'"`
		Meta string                  `xorm:"TEXT"` // store hook-specific attributes

		// HeaderAuthorizationEncrypted should be accessed using HeaderAuthorization() and SetHeaderAuthorization()
		HeaderAuthorizationEncrypted string `xorm:"TEXT"`
	}

	type ExpectedWebhook struct {
		ID                  int64 `xorm:"pk autoincr"`
		Meta                string
		HeaderAuthorization string
	}

	type HookTask struct {
		ID             int64 `xorm:"pk autoincr"`
		HookID         int64
		PayloadContent string `xorm:"LONGTEXT"`
	}

	// Prepare and load the testing database
	x, deferable := migration_tests.PrepareTestEnv(t, 0, new(Webhook), new(ExpectedWebhook), new(HookTask))
	defer deferable()
	if x == nil || t.Failed() {
		return
	}

	if err := AddHeaderAuthorizationEncryptedColWebhook(x); err != nil {
		require.NoError(t, err)
		return
	}

	expected := []ExpectedWebhook{}
	err := x.Table("expected_webhook").Asc("id").Find(&expected)
	require.NoError(t, err)

	got := []Webhook{}
	err = x.Table("webhook").Select("id, meta, header_authorization_encrypted").Asc("id").Find(&got)
	require.NoError(t, err)

	for i, e := range expected {
		assert.Equal(t, e.Meta, got[i].Meta)

		if e.HeaderAuthorization == "" {
			assert.Empty(t, got[i].HeaderAuthorizationEncrypted)
		} else {
			cipherhex := got[i].HeaderAuthorizationEncrypted
			cleartext, err := secret.DecryptSecret(setting.SecretKey, cipherhex)
			require.NoError(t, err)
			assert.Equal(t, e.HeaderAuthorization, cleartext)
		}
	}

	// ensure that no hook_task has some remaining "access_token"
	hookTasks := []HookTask{}
	err = x.Table("hook_task").Select("id, payload_content").Asc("id").Find(&hookTasks)
	require.NoError(t, err)

	for _, h := range hookTasks {
		var m map[string]any
		err := json.Unmarshal([]byte(h.PayloadContent), &m)
		require.NoError(t, err)
		assert.Nil(t, m["access_token"])
	}
}
