// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package models

import (
	"testing"

	"forgejo.org/models/db"
	"forgejo.org/models/organization"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUser_RemoveMember(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	org := unittest.AssertExistsAndLoadBean(t, &organization.Organization{ID: 3})

	// remove a user that is a member
	unittest.AssertExistsAndLoadBean(t, &organization.OrgUser{UID: 4, OrgID: 3})
	prevNumMembers := org.NumMembers
	require.NoError(t, RemoveOrgUser(db.DefaultContext, org.ID, 4))
	unittest.AssertNotExistsBean(t, &organization.OrgUser{UID: 4, OrgID: 3})
	org = unittest.AssertExistsAndLoadBean(t, &organization.Organization{ID: 3})
	assert.Equal(t, prevNumMembers-1, org.NumMembers)

	// remove a user that is not a member
	unittest.AssertNotExistsBean(t, &organization.OrgUser{UID: 5, OrgID: 3})
	prevNumMembers = org.NumMembers
	require.NoError(t, RemoveOrgUser(db.DefaultContext, org.ID, 5))
	unittest.AssertNotExistsBean(t, &organization.OrgUser{UID: 5, OrgID: 3})
	org = unittest.AssertExistsAndLoadBean(t, &organization.Organization{ID: 3})
	assert.Equal(t, prevNumMembers, org.NumMembers)

	unittest.CheckConsistencyFor(t, &user_model.User{}, &organization.Team{})
}

func TestRemoveOrgUser(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	testSuccess := func(orgID, userID int64) {
		org := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: orgID})
		expectedNumMembers := org.NumMembers
		if unittest.BeanExists(t, &organization.OrgUser{OrgID: orgID, UID: userID}) {
			expectedNumMembers--
		}
		require.NoError(t, RemoveOrgUser(db.DefaultContext, orgID, userID))
		unittest.AssertNotExistsBean(t, &organization.OrgUser{OrgID: orgID, UID: userID})
		org = unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: orgID})
		assert.Equal(t, expectedNumMembers, org.NumMembers)
	}
	testSuccess(3, 4)
	testSuccess(3, 4)

	err := RemoveOrgUser(db.DefaultContext, 7, 5)
	require.Error(t, err)
	assert.True(t, organization.IsErrLastOrgOwner(err))
	unittest.AssertExistsAndLoadBean(t, &organization.OrgUser{OrgID: 7, UID: 5})
	unittest.CheckConsistencyFor(t, &user_model.User{}, &organization.Team{})
}
