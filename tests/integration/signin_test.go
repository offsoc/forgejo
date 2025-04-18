// Copyright 2017 The Gitea Authors. All rights reserved.
// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/test"
	"forgejo.org/modules/translation"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
)

func testLoginFailed(t *testing.T, username, password, message string) {
	session := emptyTestSession(t)
	req := NewRequestWithValues(t, "POST", "/user/login", map[string]string{
		"_csrf":     GetCSRF(t, session, "/user/login"),
		"user_name": username,
		"password":  password,
	})
	resp := session.MakeRequest(t, req, http.StatusOK)

	htmlDoc := NewHTMLParser(t, resp.Body)
	resultMsg := htmlDoc.doc.Find(".ui.message>p").Text()

	assert.Equal(t, message, resultMsg)
}

func TestSignin(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	// add new user with user2's email
	user.Name = "testuser"
	user.LowerName = strings.ToLower(user.Name)
	user.ID = 0
	unittest.AssertSuccessfulInsert(t, user)

	samples := []struct {
		username string
		password string
		message  string
	}{
		{username: "wrongUsername", password: "wrongPassword", message: translation.NewLocale("en-US").TrString("form.username_password_incorrect")},
		{username: "wrongUsername", password: "password", message: translation.NewLocale("en-US").TrString("form.username_password_incorrect")},
		{username: "user15", password: "wrongPassword", message: translation.NewLocale("en-US").TrString("form.username_password_incorrect")},
		{username: "user1@example.com", password: "wrongPassword", message: translation.NewLocale("en-US").TrString("form.username_password_incorrect")},
	}

	for _, s := range samples {
		testLoginFailed(t, s.username, s.password, s.message)
	}
}

func TestSigninWithRememberMe(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	baseURL, _ := url.Parse(setting.AppURL)

	session := emptyTestSession(t)
	req := NewRequestWithValues(t, "POST", "/user/login", map[string]string{
		"_csrf":     GetCSRF(t, session, "/user/login"),
		"user_name": user.Name,
		"password":  userPassword,
		"remember":  "on",
	})
	session.MakeRequest(t, req, http.StatusSeeOther)

	c := session.GetCookie(setting.CookieRememberName)
	assert.NotNil(t, c)

	session = emptyTestSession(t)

	// Without session the settings page should not be reachable
	req = NewRequest(t, "GET", "/user/settings")
	session.MakeRequest(t, req, http.StatusSeeOther)

	req = NewRequest(t, "GET", "/user/login")
	// Set the remember me cookie for the login GET request
	session.jar.SetCookies(baseURL, []*http.Cookie{c})
	session.MakeRequest(t, req, http.StatusSeeOther)

	// With session the settings page should be reachable
	req = NewRequest(t, "GET", "/user/settings")
	session.MakeRequest(t, req, http.StatusOK)
}

func TestDisableSignin(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	// Mock alternative auth ways as enabled
	defer test.MockVariableValue(&setting.Service.EnableOpenIDSignIn, true)()
	defer test.MockVariableValue(&setting.Service.EnableOpenIDSignUp, true)()
	t.Run("Disabled", func(t *testing.T) {
		defer test.MockVariableValue(&setting.Service.EnableInternalSignIn, false)()

		t.Run("UI", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequest(t, "GET", "/user/login")
			resp := MakeRequest(t, req, http.StatusOK)
			htmlDoc := NewHTMLParser(t, resp.Body)
			htmlDoc.AssertElement(t, "form[action='/user/login']", false)
			htmlDoc.AssertElement(t, ".divider-text", false)
		})

		t.Run("Signin", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			req := NewRequest(t, "POST", "/user/login")
			MakeRequest(t, req, http.StatusForbidden)
		})
	})

	t.Run("Enabled", func(t *testing.T) {
		defer test.MockVariableValue(&setting.Service.EnableInternalSignIn, true)()

		t.Run("UI", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequest(t, "GET", "/user/login")
			resp := MakeRequest(t, req, http.StatusOK)
			htmlDoc := NewHTMLParser(t, resp.Body)
			htmlDoc.AssertElement(t, "form[action='/user/login']", true)
			htmlDoc.AssertElement(t, ".divider-text", true)
		})

		t.Run("Signin", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			req := NewRequest(t, "POST", "/user/login")
			MakeRequest(t, req, http.StatusOK)
		})
	})
}
