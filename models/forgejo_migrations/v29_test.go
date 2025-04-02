// Copyright 2025 The Forgejo Authors.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations //nolint:revive

import (
	"testing"
	"time"

	migration_tests "forgejo.org/models/migrations/test"
	"forgejo.org/modules/timeutil"

	"github.com/stretchr/testify/require"
	"xorm.io/xorm/schemas"
)

func Test_MigrateNormalizedFederatedURI(t *testing.T) {
	// Old structs
	type User struct {
		ID                     int64 `xorm:"pk autoincr"`
		NormalizedFederatedURI string
	}
	type FederatedUser struct {
		ID               int64  `xorm:"pk autoincr"`
		UserID           int64  `xorm:"NOT NULL"`
		ExternalID       string `xorm:"UNIQUE(federation_user_mapping) NOT NULL"`
		FederationHostID int64  `xorm:"UNIQUE(federation_user_mapping) NOT NULL"`
	}
	type FederationHost struct {
		ID             int64              `xorm:"pk autoincr"`
		HostFqdn       string             `xorm:"host_fqdn UNIQUE INDEX VARCHAR(255) NOT NULL"`
		SoftwareName   string             `xorm:"NOT NULL"`
		LatestActivity time.Time          `xorm:"NOT NULL"`
		Created        timeutil.TimeStamp `xorm:"created"`
		Updated        timeutil.TimeStamp `xorm:"updated"`
	}

	// Prepare TestEnv
	x, deferable := migration_tests.PrepareTestEnv(t, 0,
		new(User),
		new(FederatedUser),
		new(FederationHost),
	)
	defer deferable()
	if x == nil || t.Failed() {
		return
	}

	// test for expected results
	getColumn := func() *schemas.Column {
		tables, err := x.DBMetas()
		require.NoError(t, err)
		var table *schemas.Table
		for _, elem := range tables {
			if elem.Name == "user" {
				table = elem
				break
			}
		}
		require.Equal(t, "user", table.Name)
		return table.GetColumn("normalized_federated_uri")
	}

	require.NotNil(t, getColumn(), "NormalizedFederatedURI column exists")
	require.NoError(t, MigrateNormalizedFederatedURI(x))
	require.Nil(t, getColumn(), "NormalizedFederatedURI column was deleted")
	// idempotent
	require.NoError(t, MigrateNormalizedFederatedURI(x))

	// TODO: Find a good way to test
	// res, err := x.Query("SELECT normalized_original_url FROM federated_user WHERE user_id= ?", 3)
	// require.NoError(t, err)
	// assert.EqualValues(t, "https://my.host.x/api/activitypub/user-id/18", string(res[0]["normalized_original_url"]))
}
