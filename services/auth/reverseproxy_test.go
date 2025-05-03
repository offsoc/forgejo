// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package auth

import (
	"net/http"
	"testing"

	"forgejo.org/models/db"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/test"

	"github.com/stretchr/testify/require"
)

func TestReverseProxyAuth(t *testing.T) {
	defer test.MockVariableValue(&setting.Service.EnableReverseProxyEmail, true)()
	defer test.MockVariableValue(&setting.Service.EnableReverseProxyFullName, true)()
	defer test.MockVariableValue(&setting.Service.EnableReverseProxyFullName, true)()
	require.NoError(t, unittest.PrepareTestDatabase())

	require.NoError(t, db.TruncateBeans(db.DefaultContext, &user_model.User{}))
	require.EqualValues(t, 0, user_model.CountUsers(db.DefaultContext, nil))

	t.Run("First user should be admin", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/", nil)
		require.NoError(t, err)

		req.Header.Add(setting.ReverseProxyAuthUser, "Edgar")
		req.Header.Add(setting.ReverseProxyAuthFullName, "Edgar Allan Poe")
		req.Header.Add(setting.ReverseProxyAuthEmail, "edgar@example.org")

		rp := &ReverseProxy{}
		user := rp.newUser(req)

		require.EqualValues(t, 1, user_model.CountUsers(db.DefaultContext, nil))
		unittest.AssertExistsAndLoadBean(t, &user_model.User{Email: "edgar@example.org", Name: "Edgar", LowerName: "edgar", FullName: "Edgar Allan Poe", IsAdmin: true})
		require.Equal(t, "edgar@example.org", user.Email)
		require.Equal(t, "Edgar", user.Name)
		require.Equal(t, "edgar", user.LowerName)
		require.Equal(t, "Edgar Allan Poe", user.FullName)
		require.True(t, user.IsAdmin)
	})

	t.Run("Second user shouldn't be admin", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/", nil)
		require.NoError(t, err)

		req.Header.Add(setting.ReverseProxyAuthUser, " Gusted ")
		req.Header.Add(setting.ReverseProxyAuthFullName, "❤‿❤")
		req.Header.Add(setting.ReverseProxyAuthEmail, "gusted@example.org")

		rp := &ReverseProxy{}
		user := rp.newUser(req)

		require.EqualValues(t, 2, user_model.CountUsers(db.DefaultContext, nil))
		unittest.AssertExistsAndLoadBean(t, &user_model.User{Email: "gusted@example.org", Name: "Gusted", LowerName: "gusted", FullName: "❤‿❤"}, "is_admin = false")
		require.Equal(t, "gusted@example.org", user.Email)
		require.Equal(t, "Gusted", user.Name)
		require.Equal(t, "gusted", user.LowerName)
		require.Equal(t, "❤‿❤", user.FullName)
		require.False(t, user.IsAdmin)
	})
}
