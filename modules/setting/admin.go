// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package setting

import (
	"forgejo.org/modules/container"
)

// Admin settings
var Admin struct {
	DisableRegularOrgCreation      bool
	DefaultEmailNotification       string
	SendNotificationEmailOnNewUser bool
	UserDisabledFeatures           container.Set[string]
	ExternalUserDisableFeatures    container.Set[string]
}

func loadAdminFrom(rootCfg ConfigProvider) {
	sec := rootCfg.Section("admin")
	Admin.DisableRegularOrgCreation = sec.Key("DISABLE_REGULAR_ORG_CREATION").MustBool(false)
	Admin.DefaultEmailNotification = sec.Key("DEFAULT_EMAIL_NOTIFICATIONS").MustString("enabled")
	Admin.SendNotificationEmailOnNewUser = sec.Key("SEND_NOTIFICATION_EMAIL_ON_NEW_USER").MustBool(false)
	Admin.UserDisabledFeatures = container.SetOf(sec.Key("USER_DISABLED_FEATURES").Strings(",")...)
	Admin.ExternalUserDisableFeatures = container.SetOf(sec.Key("EXTERNAL_USER_DISABLE_FEATURES").Strings(",")...)
}

const (
	UserFeatureDeletion      = "deletion"
	UserFeatureManageSSHKeys = "manage_ssh_keys"
	UserFeatureManageGPGKeys = "manage_gpg_keys"
)
