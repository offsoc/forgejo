// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package admin

import (
	"testing"

	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/setting"
	api "forgejo.org/modules/structs"
	"forgejo.org/modules/web"
	"forgejo.org/services/contexttest"
	"forgejo.org/services/forms"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUserPost_MustChangePassword(t *testing.T) {
	unittest.PrepareTestEnv(t)
	ctx, _ := contexttest.MockContext(t, "admin/users/new")

	u := unittest.AssertExistsAndLoadBean(t, &user_model.User{
		IsAdmin: true,
		ID:      2,
	})

	ctx.Doer = u

	username := "gitea"
	email := "gitea@gitea.io"

	form := forms.AdminCreateUserForm{
		LoginType:          "local",
		LoginName:          "local",
		UserName:           username,
		Email:              email,
		Password:           "abc123ABC!=$",
		SendNotify:         false,
		MustChangePassword: true,
	}

	web.SetForm(ctx, &form)
	NewUserPost(ctx)

	assert.NotEmpty(t, ctx.Flash.SuccessMsg)

	u, err := user_model.GetUserByName(ctx, username)

	require.NoError(t, err)
	assert.Equal(t, username, u.Name)
	assert.Equal(t, email, u.Email)
	assert.True(t, u.MustChangePassword)
}

func TestNewUserPost_MustChangePasswordFalse(t *testing.T) {
	unittest.PrepareTestEnv(t)
	ctx, _ := contexttest.MockContext(t, "admin/users/new")

	u := unittest.AssertExistsAndLoadBean(t, &user_model.User{
		IsAdmin: true,
		ID:      2,
	})

	ctx.Doer = u

	username := "gitea"
	email := "gitea@gitea.io"

	form := forms.AdminCreateUserForm{
		LoginType:          "local",
		LoginName:          "local",
		UserName:           username,
		Email:              email,
		Password:           "abc123ABC!=$",
		SendNotify:         false,
		MustChangePassword: false,
	}

	web.SetForm(ctx, &form)
	NewUserPost(ctx)

	assert.NotEmpty(t, ctx.Flash.SuccessMsg)

	u, err := user_model.GetUserByName(ctx, username)

	require.NoError(t, err)
	assert.Equal(t, username, u.Name)
	assert.Equal(t, email, u.Email)
	assert.False(t, u.MustChangePassword)
}

func TestNewUserPost_InvalidEmail(t *testing.T) {
	unittest.PrepareTestEnv(t)
	ctx, _ := contexttest.MockContext(t, "admin/users/new")

	u := unittest.AssertExistsAndLoadBean(t, &user_model.User{
		IsAdmin: true,
		ID:      2,
	})

	ctx.Doer = u

	username := "gitea"
	email := "gitea@gitea.io\r\n"

	form := forms.AdminCreateUserForm{
		LoginType:          "local",
		LoginName:          "local",
		UserName:           username,
		Email:              email,
		Password:           "abc123ABC!=$",
		SendNotify:         false,
		MustChangePassword: false,
	}

	web.SetForm(ctx, &form)
	NewUserPost(ctx)

	assert.NotEmpty(t, ctx.Flash.ErrorMsg)
}

func TestNewUserPost_VisibilityDefaultPublic(t *testing.T) {
	unittest.PrepareTestEnv(t)
	ctx, _ := contexttest.MockContext(t, "admin/users/new")

	u := unittest.AssertExistsAndLoadBean(t, &user_model.User{
		IsAdmin: true,
		ID:      2,
	})

	ctx.Doer = u

	username := "gitea"
	email := "gitea@gitea.io"

	form := forms.AdminCreateUserForm{
		LoginType:          "local",
		LoginName:          "local",
		UserName:           username,
		Email:              email,
		Password:           "abc123ABC!=$",
		SendNotify:         false,
		MustChangePassword: false,
	}

	web.SetForm(ctx, &form)
	NewUserPost(ctx)

	assert.NotEmpty(t, ctx.Flash.SuccessMsg)

	u, err := user_model.GetUserByName(ctx, username)

	require.NoError(t, err)
	assert.Equal(t, username, u.Name)
	assert.Equal(t, email, u.Email)
	// As default user visibility
	assert.Equal(t, setting.Service.DefaultUserVisibilityMode, u.Visibility)
}

func TestNewUserPost_VisibilityPrivate(t *testing.T) {
	unittest.PrepareTestEnv(t)
	ctx, _ := contexttest.MockContext(t, "admin/users/new")

	u := unittest.AssertExistsAndLoadBean(t, &user_model.User{
		IsAdmin: true,
		ID:      2,
	})

	ctx.Doer = u

	username := "gitea"
	email := "gitea@gitea.io"

	form := forms.AdminCreateUserForm{
		LoginType:          "local",
		LoginName:          "local",
		UserName:           username,
		Email:              email,
		Password:           "abc123ABC!=$",
		SendNotify:         false,
		MustChangePassword: false,
		Visibility:         api.VisibleTypePrivate,
	}

	web.SetForm(ctx, &form)
	NewUserPost(ctx)

	assert.NotEmpty(t, ctx.Flash.SuccessMsg)

	u, err := user_model.GetUserByName(ctx, username)

	require.NoError(t, err)
	assert.Equal(t, username, u.Name)
	assert.Equal(t, email, u.Email)
	// As default user visibility
	assert.True(t, u.Visibility.IsPrivate())
}
