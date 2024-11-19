// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package setting

// Quota settings
var Quota = struct {
	Enabled       bool     `ini:"ENABLED"`
	DefaultGroups []string `ini:"DEFAULT_GROUPS"`

	Default struct {
		Total int64
	} `ini:"quota.default"`
}{
	Enabled:       false,
	DefaultGroups: []string{},
	Default: struct {
		Total int64
	}{
		Total: -1,
	},
}

func loadQuotaFrom(rootCfg ConfigProvider) {
	mustMapSetting(rootCfg, "quota", &Quota)
}
