// Copyright 2025 The Forgejo Authors.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations //nolint:revive

import (
	"testing"
	"time"

	migration_tests "forgejo.org/models/migrations/test"
	"forgejo.org/modules/log"
	ft "forgejo.org/modules/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_AddFederatedUserActivityTables(t *testing.T) {
	lc, cl := ft.NewLogChecker(log.DEFAULT, log.WARN)
	lc.Filter("migration[33]")
	defer cl()

	// intentionally conflicting definition
	type FederatedUser struct {
		ID     int64 `xorm:"pk autoincr"`
		UserID string
	}

	// Prepare TestEnv
	x, deferable := migration_tests.PrepareTestEnv(t, 0,
		new(FederatedUser),
	)
	sessTest := x.NewSession()
	sessTest.Insert(FederatedUser{UserID: "1234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890" +
		"1234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890" +
		"1234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890"})
	sessTest.Commit()
	defer deferable()
	if x == nil || t.Failed() {
		return
	}

	require.NoError(t, AddFederatedUserActivityTables(x))
	logFiltered, _ := lc.Check(5 * time.Second)
	assert.True(t, len(logFiltered) > 0)
}
