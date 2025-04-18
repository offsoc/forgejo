// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"testing"

	"forgejo.org/models/perm"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/test"

	"github.com/stretchr/testify/assert"
)

func TestActionsConfig(t *testing.T) {
	cfg := &ActionsConfig{}
	cfg.DisableWorkflow("test1.yaml")
	assert.Equal(t, []string{"test1.yaml"}, cfg.DisabledWorkflows)

	cfg.DisableWorkflow("test1.yaml")
	assert.Equal(t, []string{"test1.yaml"}, cfg.DisabledWorkflows)

	cfg.EnableWorkflow("test1.yaml")
	assert.Equal(t, []string{}, cfg.DisabledWorkflows)

	cfg.EnableWorkflow("test1.yaml")
	assert.Equal(t, []string{}, cfg.DisabledWorkflows)

	cfg.DisableWorkflow("test1.yaml")
	cfg.DisableWorkflow("test2.yaml")
	cfg.DisableWorkflow("test3.yaml")
	assert.Equal(t, "test1.yaml,test2.yaml,test3.yaml", cfg.ToString())
}

func TestRepoUnitAccessMode(t *testing.T) {
	assert.Equal(t, perm.AccessModeNone, UnitAccessModeNone.ToAccessMode(perm.AccessModeAdmin))
	assert.Equal(t, perm.AccessModeRead, UnitAccessModeRead.ToAccessMode(perm.AccessModeAdmin))
	assert.Equal(t, perm.AccessModeWrite, UnitAccessModeWrite.ToAccessMode(perm.AccessModeAdmin))
	assert.Equal(t, perm.AccessModeRead, UnitAccessModeUnset.ToAccessMode(perm.AccessModeRead))
}

func TestRepoPRIsUpdateStyleAllowed(t *testing.T) {
	var cfg PullRequestsConfig
	cfg = PullRequestsConfig{
		AllowRebaseUpdate: true,
	}
	assert.True(t, cfg.IsUpdateStyleAllowed(UpdateStyleMerge))
	assert.True(t, cfg.IsUpdateStyleAllowed(UpdateStyleRebase))

	cfg = PullRequestsConfig{
		AllowRebaseUpdate: false,
	}
	assert.True(t, cfg.IsUpdateStyleAllowed(UpdateStyleMerge))
	assert.False(t, cfg.IsUpdateStyleAllowed(UpdateStyleRebase))
}

func TestRepoPRGetDefaultUpdateStyle(t *testing.T) {
	defer test.MockVariableValue(&setting.Repository.PullRequest.DefaultUpdateStyle, "merge")()

	var cfg PullRequestsConfig
	cfg = PullRequestsConfig{
		DefaultUpdateStyle: "",
	}
	assert.Equal(t, UpdateStyleMerge, cfg.GetDefaultUpdateStyle())
	cfg = PullRequestsConfig{
		DefaultUpdateStyle: "rebase",
	}
	assert.Equal(t, UpdateStyleRebase, cfg.GetDefaultUpdateStyle())
	cfg = PullRequestsConfig{
		DefaultUpdateStyle: "merge",
	}
	assert.Equal(t, UpdateStyleMerge, cfg.GetDefaultUpdateStyle())

	setting.Repository.PullRequest.DefaultUpdateStyle = "rebase"
	cfg = PullRequestsConfig{
		DefaultUpdateStyle: "",
	}
	assert.Equal(t, UpdateStyleRebase, cfg.GetDefaultUpdateStyle())
	cfg = PullRequestsConfig{
		DefaultUpdateStyle: "rebase",
	}
	assert.Equal(t, UpdateStyleRebase, cfg.GetDefaultUpdateStyle())
	cfg = PullRequestsConfig{
		DefaultUpdateStyle: "merge",
	}
	assert.Equal(t, UpdateStyleMerge, cfg.GetDefaultUpdateStyle())
}
