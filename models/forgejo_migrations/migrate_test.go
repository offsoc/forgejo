// Copyright 2023 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgejo_migrations //nolint:revive

import (
	"testing"

	migration_tests "code.gitea.io/gitea/models/migrations/test"

	"github.com/stretchr/testify/assert"
)

// TestEnsureUpToDate tests the behavior of EnsureUpToDate.
func TestEnsureUpToDate(t *testing.T) {
	x, deferable := migration_tests.PrepareTestEnv(t, 0, new(ForgejoVersion))
	defer deferable()
	if x == nil || t.Failed() {
		return
	}

	// Ensure error if there's no row in Forgejo Version.
	err := EnsureUpToDate(x)
	assert.Error(t, err)

	// Insert 'good' Forgejo Version row.
	_, err = x.InsertOne(&ForgejoVersion{ID: 1, Version: ExpectedVersion()})
	assert.NoError(t, err)

	err = EnsureUpToDate(x)
	assert.NoError(t, err)

	// Modify forgejo version to have a lower version.
	_, err = x.Exec("UPDATE `forgejo_version` SET version = ? WHERE id = 1", ExpectedVersion()-1)
	assert.NoError(t, err)

	err = EnsureUpToDate(x)
	assert.Error(t, err)
}
