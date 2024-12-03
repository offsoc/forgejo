// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"code.gitea.io/gitea/models"
	"code.gitea.io/gitea/models/auth"
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/organization"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/timeutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	unittest.MainTest(m)
}

func TestDeleteUser(t *testing.T) {
	test := func(userID int64) {
		require.NoError(t, unittest.PrepareTestDatabase())
		user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: userID})

		ownedRepos := make([]*repo_model.Repository, 0, 10)
		require.NoError(t, db.GetEngine(db.DefaultContext).Find(&ownedRepos, &repo_model.Repository{OwnerID: userID}))
		if len(ownedRepos) > 0 {
			err := DeleteUser(db.DefaultContext, user, false)
			require.Error(t, err)
			assert.True(t, models.IsErrUserOwnRepos(err))
			return
		}

		orgUsers := make([]*organization.OrgUser, 0, 10)
		require.NoError(t, db.GetEngine(db.DefaultContext).Find(&orgUsers, &organization.OrgUser{UID: userID}))
		for _, orgUser := range orgUsers {
			if err := models.RemoveOrgUser(db.DefaultContext, orgUser.OrgID, orgUser.UID); err != nil {
				assert.True(t, organization.IsErrLastOrgOwner(err))
				return
			}
		}
		require.NoError(t, DeleteUser(db.DefaultContext, user, false))
		unittest.AssertNotExistsBean(t, &user_model.User{ID: userID})
		unittest.CheckConsistencyFor(t, &user_model.User{}, &repo_model.Repository{})
	}
	test(2)
	test(4)
	test(8)
	test(11)

	org := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 3})
	require.Error(t, DeleteUser(db.DefaultContext, org, false))
}

func TestPurgeUser(t *testing.T) {
	test := func(userID int64) {
		require.NoError(t, unittest.PrepareTestDatabase())
		user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: userID})

		err := DeleteUser(db.DefaultContext, user, true)
		require.NoError(t, err)

		unittest.AssertNotExistsBean(t, &user_model.User{ID: userID})
		unittest.CheckConsistencyFor(t, &user_model.User{}, &repo_model.Repository{})
	}
	test(2)
	test(4)
	test(8)
	test(11)

	org := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 3})
	require.Error(t, DeleteUser(db.DefaultContext, org, false))
}

func TestCreateUser(t *testing.T) {
	user := &user_model.User{
		Name:               "GiteaBot",
		Email:              "GiteaBot@gitea.io",
		Passwd:             ";p['////..-++']",
		IsAdmin:            false,
		Theme:              setting.UI.DefaultTheme,
		MustChangePassword: false,
	}

	require.NoError(t, user_model.CreateUser(db.DefaultContext, user))

	require.NoError(t, DeleteUser(db.DefaultContext, user, false))
}

func TestRenameUser(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 21})

	t.Run("Non-Local", func(t *testing.T) {
		u := &user_model.User{
			Type:      user_model.UserTypeIndividual,
			LoginType: auth.OAuth2,
		}
		require.ErrorIs(t, RenameUser(db.DefaultContext, u, "user_rename"), user_model.ErrUserIsNotLocal{})
	})

	t.Run("Same username", func(t *testing.T) {
		require.NoError(t, RenameUser(db.DefaultContext, user, user.Name))
	})

	t.Run("Non usable username", func(t *testing.T) {
		usernames := []string{"--diff", ".well-known", "gitea-actions", "aaa.atom", "aa.png"}
		for _, username := range usernames {
			require.Error(t, user_model.IsUsableUsername(username), "non-usable username: %s", username)
			require.Error(t, RenameUser(db.DefaultContext, user, username), "non-usable username: %s", username)
		}
	})

	t.Run("Only capitalization", func(t *testing.T) {
		caps := strings.ToUpper(user.Name)
		unittest.AssertNotExistsBean(t, &user_model.User{ID: user.ID, Name: caps})
		unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{OwnerID: user.ID, OwnerName: user.Name})

		require.NoError(t, RenameUser(db.DefaultContext, user, caps))

		unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: user.ID, Name: caps})
		unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{OwnerID: user.ID, OwnerName: caps})
	})

	t.Run("Already exists", func(t *testing.T) {
		existUser := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})

		require.ErrorIs(t, RenameUser(db.DefaultContext, user, existUser.Name), user_model.ErrUserAlreadyExist{Name: existUser.Name})
		require.ErrorIs(t, RenameUser(db.DefaultContext, user, existUser.LowerName), user_model.ErrUserAlreadyExist{Name: existUser.LowerName})
		newUsername := fmt.Sprintf("uSEr%d", existUser.ID)
		require.ErrorIs(t, RenameUser(db.DefaultContext, user, newUsername), user_model.ErrUserAlreadyExist{Name: newUsername})
	})

	t.Run("Normal", func(t *testing.T) {
		oldUsername := user.Name
		newUsername := "User_Rename"

		require.NoError(t, RenameUser(db.DefaultContext, user, newUsername))
		unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: user.ID, Name: newUsername, LowerName: strings.ToLower(newUsername)})

		redirectUID, err := user_model.LookupUserRedirect(db.DefaultContext, oldUsername)
		require.NoError(t, err)
		assert.EqualValues(t, user.ID, redirectUID)

		unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{OwnerID: user.ID, OwnerName: user.Name})
	})
}

func TestCreateUser_Issue5882(t *testing.T) {
	// Init settings
	_ = setting.Admin

	passwd := ".//.;1;;//.,-=_"

	tt := []struct {
		user               *user_model.User
		disableOrgCreation bool
	}{
		{&user_model.User{Name: "GiteaBot", Email: "GiteaBot@gitea.io", Passwd: passwd, MustChangePassword: false}, false},
		{&user_model.User{Name: "GiteaBot2", Email: "GiteaBot2@gitea.io", Passwd: passwd, MustChangePassword: false}, true},
	}

	setting.Service.DefaultAllowCreateOrganization = true

	for _, v := range tt {
		setting.Admin.DisableRegularOrgCreation = v.disableOrgCreation

		require.NoError(t, user_model.CreateUser(db.DefaultContext, v.user))

		u, err := user_model.GetUserByEmail(db.DefaultContext, v.user.Email)
		require.NoError(t, err)

		assert.Equal(t, !u.AllowCreateOrganization, v.disableOrgCreation)

		require.NoError(t, DeleteUser(db.DefaultContext, v.user, false))
	}
}

func TestDeleteInactiveUsers(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	// Add an inactive user older than a minute, with an associated email_address record.
	oldUser := &user_model.User{Name: "OldInactive", LowerName: "oldinactive", Email: "old@example.com", CreatedUnix: timeutil.TimeStampNow().Add(-120)}
	_, err := db.GetEngine(db.DefaultContext).NoAutoTime().Insert(oldUser)
	require.NoError(t, err)
	oldEmail := &user_model.EmailAddress{UID: oldUser.ID, IsPrimary: true, Email: "old@example.com", LowerEmail: "old@example.com"}
	err = db.Insert(db.DefaultContext, oldEmail)
	require.NoError(t, err)

	// Add an inactive user that's not older than a minute, with an associated email_address record.
	newUser := &user_model.User{Name: "NewInactive", LowerName: "newinactive", Email: "new@example.com"}
	err = db.Insert(db.DefaultContext, newUser)
	require.NoError(t, err)
	newEmail := &user_model.EmailAddress{UID: newUser.ID, IsPrimary: true, Email: "new@example.com", LowerEmail: "new@example.com"}
	err = db.Insert(db.DefaultContext, newEmail)
	require.NoError(t, err)

	err = DeleteInactiveUsers(db.DefaultContext, time.Minute)
	require.NoError(t, err)

	// User older than a minute should be deleted along with their email address.
	unittest.AssertExistsIf(t, false, oldUser)
	unittest.AssertExistsIf(t, false, oldEmail)

	// User not older than a minute shouldn't be deleted and their emaill address should still exist.
	unittest.AssertExistsIf(t, true, newUser)
	unittest.AssertExistsIf(t, true, newEmail)
}
