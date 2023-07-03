// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package setting

import (
	"fmt"
	"strings"
)

// Actions settings
var (
	Actions = struct {
		LogStorage            *Storage // how the created logs should be stored
		ArtifactStorage       *Storage // how the created artifacts should be stored
		ArtifactRetentionDays int64    `ini:"ARTIFACT_RETENTION_DAYS"`
		Enabled               bool
		DefaultActionsURL     defaultActionsURL `ini:"DEFAULT_ACTIONS_URL"`
	}{
		Enabled:           false,
		DefaultActionsURL: defaultActionsURLForgejo,
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

	actionsSec, _ := rootCfg.GetSection("actions.artifacts")

	Actions.ArtifactStorage, err = getStorage(rootCfg, "actions_artifacts", "", actionsSec)

	// default to 90 days in Github Actions
	if Actions.ArtifactRetentionDays <= 0 {
		Actions.ArtifactRetentionDays = 90
	}

	return err
}
