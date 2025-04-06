// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issues_test

import (
	"testing"

	issues_model "forgejo.org/models/issues"
	"forgejo.org/models/unittest"

	_ "forgejo.org/models"
	_ "forgejo.org/models/actions"
	_ "forgejo.org/models/activities"
	_ "forgejo.org/models/repo"
	_ "forgejo.org/models/user"

	"github.com/stretchr/testify/require"
)

func TestFixturesAreConsistent(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	unittest.CheckConsistencyFor(t,
		&issues_model.Issue{},
		&issues_model.PullRequest{},
		&issues_model.Milestone{},
		&issues_model.Label{},
	)
}

func TestMain(m *testing.M) {
	unittest.MainTest(m)
}
