// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgejo_migrations //nolint:revive

import (
	"testing"

	"forgejo.org/models/auth"
	migration_tests "forgejo.org/models/migrations/test"
	"forgejo.org/modules/keying"
	"forgejo.org/modules/timeutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_MigrateTwoFactorToKeying(t *testing.T) {
	type TwoFactor struct { //revive:disable-line:exported
		ID               int64 `xorm:"pk autoincr"`
		UID              int64 `xorm:"UNIQUE"`
		Secret           string
		ScratchSalt      string
		ScratchHash      string
		LastUsedPasscode string             `xorm:"VARCHAR(10)"`
		CreatedUnix      timeutil.TimeStamp `xorm:"INDEX created"`
		UpdatedUnix      timeutil.TimeStamp `xorm:"INDEX updated"`
	}

	// Prepare and load the testing database
	x, deferable := migration_tests.PrepareTestEnv(t, 0, new(TwoFactor))
	defer deferable()
	if x == nil || t.Failed() {
		return
	}

	cnt, err := x.Table("two_factor").Count()
	require.NoError(t, err)
	assert.EqualValues(t, 2, cnt)

	require.NoError(t, MigrateTwoFactorToKeying(x))

	cnt, err = x.Table("two_factor").Count()
	require.NoError(t, err)
	assert.EqualValues(t, 1, cnt)

	var twofactor auth.TwoFactor
	_, err = x.Table("two_factor").ID(1).Get(&twofactor)
	require.NoError(t, err)

	secretBytes, err := keying.DeriveKey(keying.ContextTOTP).Decrypt(twofactor.Secret, keying.ColumnAndID("secret", twofactor.ID))
	require.NoError(t, err)
	assert.Equal(t, []byte("AVDYS32OPIAYSNBG2NKYV4AHBVEMKKKIGBQ46OXTLMJO664G4TIECOGEANMSNBLS"), secretBytes)
}
