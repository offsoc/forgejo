// Copyright 2024 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package v1_22 //nolint

import (
	"forgejo.org/modules/timeutil"
	webhook_module "forgejo.org/modules/webhook"

	"xorm.io/xorm"
)

// HookTask represents a hook task.
// exact copy of models/webhook/hooktask.go when this migration was created
//   - xorm:"-" fields deleted
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

	// Version number to allow for smooth version upgrades:
	//  - Version 1: PayloadContent contains the JSON as send to the URL
	//  - Version 2: PayloadContent contains the original event
	PayloadVersion int `xorm:"DEFAULT 1"`
}

func AddPayloadVersionToHookTaskTable(x *xorm.Engine) error {
	// create missing column
	if _, err := x.SyncWithOptions(xorm.SyncOptions{
		IgnoreIndices:    true,
		IgnoreConstrains: true,
	}, new(HookTask)); err != nil {
		return err
	}
	_, err := x.Exec("UPDATE hook_task SET payload_version = 1 WHERE payload_version IS NULL")
	return err
}
