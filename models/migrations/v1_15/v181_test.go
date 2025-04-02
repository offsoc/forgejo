// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package v1_15 //nolint

import (
	"strings"
	"testing"

	migration_tests "forgejo.org/models/migrations/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_AddPrimaryEmail2EmailAddress(t *testing.T) {
	type User struct {
		ID       int64
		Email    string
		IsActive bool
	}

	// Prepare and load the testing database
	x, deferable := migration_tests.PrepareTestEnv(t, 0, new(User))
	if x == nil || t.Failed() {
		defer deferable()
		return
	}
	defer deferable()

	err := AddPrimaryEmail2EmailAddress(x)
	require.NoError(t, err)

	type EmailAddress struct {
		ID          int64  `xorm:"pk autoincr"`
		UID         int64  `xorm:"INDEX NOT NULL"`
		Email       string `xorm:"UNIQUE NOT NULL"`
		LowerEmail  string `xorm:"UNIQUE NOT NULL"`
		IsActivated bool
		IsPrimary   bool `xorm:"DEFAULT(false) NOT NULL"`
	}

	users := make([]User, 0, 20)
	err = x.Find(&users)
	require.NoError(t, err)

	for _, user := range users {
		var emailAddress EmailAddress
		has, err := x.Where("lower_email=?", strings.ToLower(user.Email)).Get(&emailAddress)
		require.NoError(t, err)
		assert.True(t, has)
		assert.True(t, emailAddress.IsPrimary)
		assert.Equal(t, user.IsActive, emailAddress.IsActivated)
		assert.Equal(t, user.ID, emailAddress.UID)
	}
}
