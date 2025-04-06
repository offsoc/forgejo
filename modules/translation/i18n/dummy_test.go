// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package i18n_test

import (
	"testing"

	"forgejo.org/modules/translation/i18n"

	"github.com/stretchr/testify/assert"
)

func TestFormatDummy(t *testing.T) {
	assert.Equal(t, "(admin.config.git_max_diff_lines)", i18n.FormatDummy("admin.config.git_max_diff_lines"))
	assert.Equal(t, "(dashboard)", i18n.FormatDummy("dashboard"))
	assert.Equal(t, "(branch.create_branch: main)", i18n.FormatDummy("branch.create_branch", "main"))
	assert.Equal(t, "(test.test: a, 1, true)", i18n.FormatDummy("test.test", "a", 1, true))
}
