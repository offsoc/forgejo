// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"testing"

	"code.gitea.io/gitea/models/perm"

	"github.com/stretchr/testify/assert"
)

func TestActionsConfig(t *testing.T) {
	cfg := &ActionsConfig{}
	cfg.DisableWorkflow("test1.yaml")
	assert.EqualValues(t, []string{"test1.yaml"}, cfg.DisabledWorkflows)

	cfg.DisableWorkflow("test1.yaml")
	assert.EqualValues(t, []string{"test1.yaml"}, cfg.DisabledWorkflows)

	cfg.EnableWorkflow("test1.yaml")
	assert.EqualValues(t, []string{}, cfg.DisabledWorkflows)

	cfg.EnableWorkflow("test1.yaml")
	assert.EqualValues(t, []string{}, cfg.DisabledWorkflows)

	cfg.DisableWorkflow("test1.yaml")
	cfg.DisableWorkflow("test2.yaml")
	cfg.DisableWorkflow("test3.yaml")
	assert.EqualValues(t, "test1.yaml,test2.yaml,test3.yaml", cfg.ToString())
}

func TestRepoUnitAccessMode(t *testing.T) {
	assert.Equal(t, perm.AccessModeNone, UnitAccessModeNone.ToAccessMode(perm.AccessModeAdmin))
	assert.Equal(t, perm.AccessModeRead, UnitAccessModeRead.ToAccessMode(perm.AccessModeAdmin))
	assert.Equal(t, perm.AccessModeWrite, UnitAccessModeWrite.ToAccessMode(perm.AccessModeAdmin))
	assert.Equal(t, perm.AccessModeRead, UnitAccessModeUnset.ToAccessMode(perm.AccessModeRead))
}
