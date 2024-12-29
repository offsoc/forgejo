// Copyright 2024 The Gitea Authors. All rights reserved.
// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package generic_test

import (
	"testing"

	generic_packages_service "code.gitea.io/gitea/services/packages/generic"

	"github.com/stretchr/testify/assert"
)

func TestValidatePackageName(t *testing.T) {
	bad := []string{
		"",
		".",
		"..",
		"-",
		"a?b",
		"a b",
		"a/b",
	}
	for _, name := range bad {
		assert.False(t, generic_packages_service.IsValidPackageName(name), "bad=%q", name)
	}

	good := []string{
		"a",
		"1",
		"a-",
		"a_b",
		"c.d+",
	}
	for _, name := range good {
		assert.True(t, generic_packages_service.IsValidPackageName(name), "good=%q", name)
	}
}

func TestValidateFileName(t *testing.T) {
	bad := []string{
		"",
		".",
		"..",
		"a?b",
		"a/b",
		" a",
		"a ",
	}
	for _, name := range bad {
		assert.False(t, generic_packages_service.IsValidFileName(name), "bad=%q", name)
	}

	good := []string{
		"-",
		"a",
		"1",
		"a-",
		"a_b",
		"a b",
		"c.d+",
		`-_+=:;.()[]{}~!@#$%^& aA1`,
	}
	for _, name := range good {
		assert.True(t, generic_packages_service.IsValidFileName(name), "good=%q", name)
	}
}
