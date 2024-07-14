// Copyright 2024 The Forgejo Authors c/o Codeberg e.V.. All rights reserved.
// SPDX-License-Identifier: MIT

package v1_22 //nolint

import (
	"strconv"
	"testing"

	migration_tests "code.gitea.io/gitea/models/migrations/test"
	"code.gitea.io/gitea/modules/timeutil"
	webhook_module "code.gitea.io/gitea/modules/webhook"

	"github.com/stretchr/testify/assert"
)

func Test_AddPayloadVersionToHookTaskTable(t *testing.T) {
	type HookTaskMigrated HookTask

	// HookTask represents a hook task, as of before the migration
	type HookTask struct {
		ID             int64  `xorm:"pk autoincr"`
		HookID         int64  `xorm:"index"`
		UUID           string `xorm:"unique"`
		PayloadContent string `xorm:"LONGTEXT"`
		EventType      webhook_module.HookEventType
		IsDelivered    bool
		Delivered      timeutil.TimeStampNano

		// History info.
		IsSucceed       bool
		RequestContent  string `xorm:"LONGTEXT"`
		ResponseContent string `xorm:"LONGTEXT"`
	}

	// Prepare and load the testing database
	x, deferable := migration_tests.PrepareTestEnv(t, 0, new(HookTask), new(HookTaskMigrated))
	defer deferable()
	if x == nil || t.Failed() {
		return
	}

	assert.NoError(t, AddPayloadVersionToHookTaskTable(x))

	expected := []HookTaskMigrated{}
	assert.NoError(t, x.Table("hook_task_migrated").Asc("id").Find(&expected))
	assert.Len(t, expected, 2)

	got := []HookTaskMigrated{}
	assert.NoError(t, x.Table("hook_task").Asc("id").Find(&got))

	for i, expected := range expected {
		expected, got := expected, got[i]
		t.Run(strconv.FormatInt(expected.ID, 10), func(t *testing.T) {
			assert.Equal(t, expected.PayloadVersion, got.PayloadVersion)
		})
	}
}
