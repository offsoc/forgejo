// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package org

import (
	"testing"

	"forgejo.org/models"
	"forgejo.org/models/db"
	"forgejo.org/models/organization"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	unittest.MainTest(m)
}

func TestDeleteOrganization(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	org := unittest.AssertExistsAndLoadBean(t, &organization.Organization{ID: 6})
	require.NoError(t, DeleteOrganization(db.DefaultContext, org, false))
	unittest.AssertNotExistsBean(t, &organization.Organization{ID: 6})
	unittest.AssertNotExistsBean(t, &organization.OrgUser{OrgID: 6})
	unittest.AssertNotExistsBean(t, &organization.Team{OrgID: 6})

	org = unittest.AssertExistsAndLoadBean(t, &organization.Organization{ID: 3})
	err := DeleteOrganization(db.DefaultContext, org, false)
	require.Error(t, err)
	assert.True(t, models.IsErrUserOwnRepos(err))

	user := unittest.AssertExistsAndLoadBean(t, &organization.Organization{ID: 5})
	require.Error(t, DeleteOrganization(db.DefaultContext, user, false))
	unittest.CheckConsistencyFor(t, &user_model.User{}, &organization.Team{})
}
