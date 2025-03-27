// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user_test

import (
	"testing"

	"forgejo.org/models/db"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLookupUserRedirect(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	userID, err := user_model.LookupUserRedirect(db.DefaultContext, "olduser1")
	require.NoError(t, err)
	assert.EqualValues(t, 1, userID)

	_, err = user_model.LookupUserRedirect(db.DefaultContext, "doesnotexist")
	assert.True(t, user_model.IsErrUserRedirectNotExist(err))
}
