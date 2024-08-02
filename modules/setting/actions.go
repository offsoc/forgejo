// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package setting

import (
	"fmt"
	"strings"
	"time"
)

// Actions settings
var (
	Actions = struct {
		Enabled               bool
		LogStorage            *Storage          // how the created logs should be stored
		LogRetentionDays      int64             `ini:"LOG_RETENTION_DAYS"`
		ArtifactStorage       *Storage          // how the created artifacts should be stored
		ArtifactRetentionDays int64             `ini:"ARTIFACT_RETENTION_DAYS"`
		DefaultActionsURL     defaultActionsURL `ini:"DEFAULT_ACTIONS_URL"`
		ZombieTaskTimeout     time.Duration     `ini:"ZOMBIE_TASK_TIMEOUT"`
		EndlessTaskTimeout    time.Duration     `ini:"ENDLESS_TASK_TIMEOUT"`
		AbandonedJobTimeout   time.Duration     `ini:"ABANDONED_JOB_TIMEOUT"`
		SkipWorkflowStrings   []string          `ìni:"SKIP_WORKFLOW_STRINGS"`
		LimitDispatchInputs   int64             `ini:"LIMIT_DISPATCH_INPUTS"`
	}{
		Enabled:             true,
		DefaultActionsURL:   defaultActionsURLForgejo,
		SkipWorkflowStrings: []string{"[skip ci]", "[ci skip]", "[no ci]", "[skip actions]", "[actions skip]"},
		LimitDispatchInputs: 10,
	}
)

type defaultActionsURL string

func (url defaultActionsURL) URL() string {
	switch url {
	case defaultActionsURLGitHub:
		return "https://github.com"
	case defaultActionsURLSelf:
		return strings.TrimSuffix(AppURL, "/")
	default:
		return string(url)
	}
}

const (
	defaultActionsURLForgejo = "https://code.forgejo.org"
	defaultActionsURLGitHub  = "github" // https://github.com
	defaultActionsURLSelf    = "self"   // the root URL of the self-hosted instance
)

func loadActionsFrom(rootCfg ConfigProvider) error {
	sec := rootCfg.Section("actions")
	err := sec.MapTo(&Actions)
	if err != nil {
		return fmt.Errorf("failed to map Actions settings: %v", err)
	}

	// don't support to read configuration from [actions]
	Actions.LogStorage, err = getStorage(rootCfg, "actions_log", "", nil)
	if err != nil {
		return err
	}
	// default to 1 year
	if Actions.LogRetentionDays <= 0 {
		Actions.LogRetentionDays = 365
	}

	actionsSec, _ := rootCfg.GetSection("actions.artifacts")

	Actions.ArtifactStorage, err = getStorage(rootCfg, "actions_artifacts", "", actionsSec)
	if err != nil {
		return err
	}

	// default to 90 days in Github Actions
	if Actions.ArtifactRetentionDays <= 0 {
		Actions.ArtifactRetentionDays = 90
	}

	Actions.ZombieTaskTimeout = sec.Key("ZOMBIE_TASK_TIMEOUT").MustDuration(10 * time.Minute)
	Actions.EndlessTaskTimeout = sec.Key("ENDLESS_TASK_TIMEOUT").MustDuration(3 * time.Hour)
	Actions.AbandonedJobTimeout = sec.Key("ABANDONED_JOB_TIMEOUT").MustDuration(24 * time.Hour)

	return nil
}
