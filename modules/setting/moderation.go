// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package setting

// Moderation settings
var Moderation = struct {
	Enabled bool `ini:"ENABLED"`
}{
	Enabled: false,
}

func loadModerationFrom(rootCfg ConfigProvider) {
	mustMapSetting(rootCfg, "moderation", &Moderation)
}
