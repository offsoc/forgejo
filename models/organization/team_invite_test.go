// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package organization_test

import (
	"testing"

	"forgejo.org/models/db"
	"forgejo.org/models/organization"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTeamInvite(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	team := unittest.AssertExistsAndLoadBean(t, &organization.Team{ID: 2})

	t.Run("MailExistsInTeam", func(t *testing.T) {
		user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

		// user 2 already added to team 2, should result in error
		_, err := organization.CreateTeamInvite(db.DefaultContext, user2, team, user2.Email)
		require.Error(t, err)
	})

	t.Run("CreateAndRemove", func(t *testing.T) {
		user1 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})

		invite, err := organization.CreateTeamInvite(db.DefaultContext, user1, team, "org3@example.com")
		assert.NotNil(t, invite)
		require.NoError(t, err)

		// Shouldn't allow duplicate invite
		_, err = organization.CreateTeamInvite(db.DefaultContext, user1, team, "org3@example.com")
		require.Error(t, err)

		// should remove invite
		require.NoError(t, organization.RemoveInviteByID(db.DefaultContext, invite.ID, invite.TeamID))

		// invite should not exist
		_, err = organization.GetInviteByToken(db.DefaultContext, invite.Token)
		require.Error(t, err)
	})
}
