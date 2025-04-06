// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"database/sql"
	"testing"

	"forgejo.org/models/db"
	"forgejo.org/models/forgefed"
	"forgejo.org/models/user"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStoreFederationHost(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	t.Run("ExplicitNull", func(t *testing.T) {
		federationHost := forgefed.FederationHost{
			HostFqdn: "ExplicitNull",
			// Explicit null on KeyID and PublicKey
			KeyID:     sql.NullString{Valid: false},
			PublicKey: sql.Null[sql.RawBytes]{Valid: false},
		}

		_, err := db.GetEngine(db.DefaultContext).Insert(&federationHost)
		require.NoError(t, err)

		dbFederationHost := new(forgefed.FederationHost)
		has, err := db.GetEngine(db.DefaultContext).Where("host_fqdn=?", "ExplicitNull").Get(dbFederationHost)
		require.NoError(t, err)
		assert.True(t, has)

		assert.False(t, dbFederationHost.KeyID.Valid)
		assert.False(t, dbFederationHost.PublicKey.Valid)
	})

	t.Run("NotNull", func(t *testing.T) {
		federationHost := forgefed.FederationHost{
			HostFqdn:  "ImplicitNull",
			KeyID:     sql.NullString{Valid: true, String: "meow"},
			PublicKey: sql.Null[sql.RawBytes]{Valid: true, V: sql.RawBytes{0x23, 0x42}},
		}

		_, err := db.GetEngine(db.DefaultContext).Insert(&federationHost)
		require.NoError(t, err)

		dbFederationHost := new(forgefed.FederationHost)
		has, err := db.GetEngine(db.DefaultContext).Where("host_fqdn=?", "ImplicitNull").Get(dbFederationHost)
		require.NoError(t, err)
		assert.True(t, has)

		assert.True(t, dbFederationHost.KeyID.Valid)
		assert.Equal(t, "meow", dbFederationHost.KeyID.String)

		assert.True(t, dbFederationHost.PublicKey.Valid)
		assert.Equal(t, sql.RawBytes{0x23, 0x42}, dbFederationHost.PublicKey.V)
	})
}

func TestStoreFederatedUser(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	t.Run("ExplicitNull", func(t *testing.T) {
		federatedUser := user.FederatedUser{
			UserID:           0,
			ExternalID:       "ExplicitNull",
			FederationHostID: 0,
			KeyID:            sql.NullString{Valid: false},
			PublicKey:        sql.Null[sql.RawBytes]{Valid: false},
		}

		_, err := db.GetEngine(db.DefaultContext).Insert(&federatedUser)
		require.NoError(t, err)

		dbFederatedUser := new(user.FederatedUser)
		has, err := db.GetEngine(db.DefaultContext).Where("user_id=?", 0).Get(dbFederatedUser)
		require.NoError(t, err)
		assert.True(t, has)

		assert.False(t, dbFederatedUser.KeyID.Valid)
		assert.False(t, dbFederatedUser.PublicKey.Valid)
	})

	t.Run("NotNull", func(t *testing.T) {
		federatedUser := user.FederatedUser{
			UserID:           1,
			ExternalID:       "ImplicitNull",
			FederationHostID: 1,
			KeyID:            sql.NullString{Valid: true, String: "woem"},
			PublicKey:        sql.Null[sql.RawBytes]{Valid: true, V: sql.RawBytes{0x42, 0x23}},
		}

		_, err := db.GetEngine(db.DefaultContext).Insert(&federatedUser)
		require.NoError(t, err)

		dbFederatedUser := new(user.FederatedUser)
		has, err := db.GetEngine(db.DefaultContext).Where("user_id=?", 1).Get(dbFederatedUser)
		require.NoError(t, err)
		assert.True(t, has)

		assert.True(t, dbFederatedUser.KeyID.Valid)
		assert.Equal(t, "woem", dbFederatedUser.KeyID.String)
		assert.True(t, dbFederatedUser.PublicKey.Valid)
		assert.Equal(t, sql.RawBytes{0x42, 0x23}, dbFederatedUser.PublicKey.V)
	})
}
