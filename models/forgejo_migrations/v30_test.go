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
	getColumn := func(tn, co string) *schemas.Column {
		tables, err := x.DBMetas()
		require.NoError(t, err)
		var table *schemas.Table
		for _, elem := range tables {
			if elem.Name == tn {
				table = elem
				break
			}
		}
		return table.GetColumn(co)
	}

	require.NotNil(t, getColumn("user", "normalized_federated_uri"))
	require.Nil(t, getColumn("federation_host", "host_port"))
	require.Nil(t, getColumn("federation_host", "host_schema"))
	cnt1, err := x.Table("federated_user").Count()
	require.NoError(t, err)
	require.Equal(t, int64(2), cnt1)

	require.NoError(t, MigrateNormalizedFederatedURI(x))

	require.Nil(t, getColumn("user", "normalized_federated_uri"))
	require.NotNil(t, getColumn("federation_host", "host_port"))
	require.NotNil(t, getColumn("federation_host", "host_schema"))
	cnt2, err := x.Table("federated_user").Count()
	require.NoError(t, err)
	require.Equal(t, int64(1), cnt2)

	// idempotent
	require.NoError(t, MigrateNormalizedFederatedURI(x))
}
