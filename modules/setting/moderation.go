// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package setting

import (
	"fmt"
	"time"
)

// Moderation settings
var Moderation = struct {
	Enabled                      bool          `ini:"ENABLED"`
	RemoveResolvedReportsTimeout time.Duration `ini:"REMOVE_RESOLVED_REPORTS_TIMEOUT"`
}{
	Enabled: false,
}

func loadModerationFrom(rootCfg ConfigProvider) error {
	sec := rootCfg.Section("actions")
	err := sec.MapTo(&Moderation)
	if err != nil {
		return fmt.Errorf("failed to map Actions settings: %v", err)
	}

	Moderation.RemoveResolvedReportsTimeout = sec.Key("REMOVE_RESOLVED_REPORTS_TIMEOUT").MustDuration(10 * time.Minute)
	return nil
}
