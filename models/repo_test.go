// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package models

import (
	"testing"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/unittest"

	"github.com/stretchr/testify/require"
)

func TestCheckRepoStats(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	require.NoError(t, CheckRepoStats(db.DefaultContext))
}

func TestDoctorUserStarNum(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	require.NoError(t, DoctorUserStarNum(db.DefaultContext))
}
