// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package setting

import (
	"testing"

	"forgejo.org/modules/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitConfig(t *testing.T) {
	defer test.MockProtect(&Git)()
	defer test.MockProtect(&GitConfig)()

	cfg, err := NewConfigProviderFromData(`
[git.config]
a.b = 1
`)
	require.NoError(t, err)
	loadGitFrom(cfg)
	assert.Equal(t, "1", GitConfig.Options["a.b"])
	assert.Equal(t, "histogram", GitConfig.Options["diff.algorithm"])

	cfg, err = NewConfigProviderFromData(`
[git.config]
diff.algorithm = other
`)
	require.NoError(t, err)
	loadGitFrom(cfg)
	assert.Equal(t, "other", GitConfig.Options["diff.algorithm"])
}

func TestGitReflog(t *testing.T) {
	defer test.MockProtect(&Git)()
	defer test.MockProtect(&GitConfig)()

	// default reflog config without legacy options
	cfg, err := NewConfigProviderFromData(``)
	require.NoError(t, err)
	loadGitFrom(cfg)

	assert.Equal(t, "true", GitConfig.GetOption("core.logAllRefUpdates"))
	assert.Equal(t, "90", GitConfig.GetOption("gc.reflogExpire"))

	// custom reflog config by legacy options
	cfg, err = NewConfigProviderFromData(`
[git.reflog]
ENABLED = false
EXPIRATION = 123
`)
	require.NoError(t, err)
	loadGitFrom(cfg)

	assert.Equal(t, "false", GitConfig.GetOption("core.logAllRefUpdates"))
	assert.Equal(t, "123", GitConfig.GetOption("gc.reflogExpire"))
}
