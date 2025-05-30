// Copyright 2017 The Gitea Authors. All rights reserved.
// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/cache"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/test"
	"forgejo.org/modules/translation"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
)

func TestSignup(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	defer test.MockVariableValue(&setting.Service.EnableCaptcha, false)()

	req := NewRequestWithValues(t, "POST", "/user/sign_up", map[string]string{
		"user_name": "exampleUser",
		"email":     "exampleUser@example.com",
		"password":  "examplePassword!1",
		"retype":    "examplePassword!1",
	})
	MakeRequest(t, req, http.StatusSeeOther)

	// should be able to view new user's page
	req = NewRequest(t, "GET", "/exampleUser")
	MakeRequest(t, req, http.StatusOK)
}

func TestSignupAsRestricted(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	defer test.MockVariableValue(&setting.Service.EnableCaptcha, false)()
	defer test.MockVariableValue(&setting.Service.DefaultUserIsRestricted, true)()

	req := NewRequestWithValues(t, "POST", "/user/sign_up", map[string]string{
		"user_name": "restrictedUser",
		"email":     "restrictedUser@example.com",
		"password":  "examplePassword!1",
		"retype":    "examplePassword!1",
	})
	MakeRequest(t, req, http.StatusSeeOther)

	// should be able to view new user's page
	req = NewRequest(t, "GET", "/restrictedUser")
	MakeRequest(t, req, http.StatusOK)

	user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: "restrictedUser"})
	assert.True(t, user2.IsRestricted)
}

func TestSignupEmail(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	defer test.MockVariableValue(&setting.Service.EnableCaptcha, false)()

	tests := []struct {
		email      string
		wantStatus int
		wantMsg    string
	}{
		{"exampleUser@example.com\r\n", http.StatusOK, translation.NewLocale("en-US").TrString("form.email_invalid")},
		{"exampleUser@example.com\r", http.StatusOK, translation.NewLocale("en-US").TrString("form.email_invalid")},
		{"exampleUser@example.com\n", http.StatusOK, translation.NewLocale("en-US").TrString("form.email_invalid")},
		{"exampleUser@example.com", http.StatusSeeOther, ""},
	}

	for i, test := range tests {
		req := NewRequestWithValues(t, "POST", "/user/sign_up", map[string]string{
			"user_name": fmt.Sprintf("exampleUser%d", i),
			"email":     test.email,
			"password":  "examplePassword!1",
			"retype":    "examplePassword!1",
		})
		resp := MakeRequest(t, req, test.wantStatus)
		if test.wantMsg != "" {
			htmlDoc := NewHTMLParser(t, resp.Body)
			assert.Equal(t,
				test.wantMsg,
				strings.TrimSpace(htmlDoc.doc.Find(".ui.message").Text()),
			)
		}
	}
}

func TestSignupEmailChangeForInactiveUser(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// Disable the captcha & enable email confirmation for registrations
	defer test.MockVariableValue(&setting.Service.EnableCaptcha, false)()
	defer test.MockVariableValue(&setting.Service.RegisterEmailConfirm, true)()

	// Create user
	req := NewRequestWithValues(t, "POST", "/user/sign_up", map[string]string{
		"user_name": "exampleUserX",
		"email":     "wrong-email@example.com",
		"password":  "examplePassword!1",
		"retype":    "examplePassword!1",
	})
	MakeRequest(t, req, http.StatusOK)

	session := loginUserWithPassword(t, "exampleUserX", "examplePassword!1")

	// Verify that the initial e-mail is the wrong one.
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: "exampleUserX"})
	assert.Equal(t, "wrong-email@example.com", user.Email)

	// Change the email address
	req = NewRequestWithValues(t, "POST", "/user/activate", map[string]string{
		"email": "fine-email@example.com",
	})
	session.MakeRequest(t, req, http.StatusSeeOther)

	// Verify that the email was updated
	user = unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: "exampleUserX"})
	assert.Equal(t, "fine-email@example.com", user.Email)

	// Try to change the email again
	req = NewRequestWithValues(t, "POST", "/user/activate", map[string]string{
		"email": "wrong-again@example.com",
	})
	session.MakeRequest(t, req, http.StatusSeeOther)
	// Verify that the email was NOT updated
	user = unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: "exampleUserX"})
	assert.Equal(t, "fine-email@example.com", user.Email)
}

func TestSignupEmailChangeForActiveUser(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// Disable the captcha & enable email confirmation for registrations
	defer test.MockVariableValue(&setting.Service.EnableCaptcha, false)()
	defer test.MockVariableValue(&setting.Service.RegisterEmailConfirm, false)()

	// Create user
	req := NewRequestWithValues(t, "POST", "/user/sign_up", map[string]string{
		"user_name": "exampleUserY",
		"email":     "wrong-email-2@example.com",
		"password":  "examplePassword!1",
		"retype":    "examplePassword!1",
	})
	MakeRequest(t, req, http.StatusSeeOther)

	session := loginUserWithPassword(t, "exampleUserY", "examplePassword!1")

	// Verify that the initial e-mail is the wrong one.
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: "exampleUserY"})
	assert.Equal(t, "wrong-email-2@example.com", user.Email)

	// Changing the email for a validated address is not available
	req = NewRequestWithValues(t, "POST", "/user/activate", map[string]string{
		"email": "fine-email-2@example.com",
	})
	session.MakeRequest(t, req, http.StatusNotFound)

	// Verify that the email remained unchanged
	user = unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: "exampleUserY"})
	assert.Equal(t, "wrong-email-2@example.com", user.Email)
}

func TestSignupImageCaptcha(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	defer test.MockVariableValue(&setting.Service.RegisterEmailConfirm, false)()
	defer test.MockVariableValue(&setting.Service.EnableCaptcha, true)()
	defer test.MockVariableValue(&setting.Service.CaptchaType, "image")()
	c := cache.GetCache()

	req := NewRequest(t, "GET", "/user/sign_up")
	resp := MakeRequest(t, req, http.StatusOK)
	htmlDoc := NewHTMLParser(t, resp.Body)

	idCaptcha, ok := htmlDoc.Find("input[name='img-captcha-id']").Attr("value")
	assert.True(t, ok)

	digits, ok := c.Get("captcha:" + idCaptcha).(string)
	assert.True(t, ok)
	assert.Len(t, digits, 6)

	digitStr := ""
	// Convert digits to ASCII digits.
	for _, digit := range digits {
		digitStr += string(digit + '0')
	}

	req = NewRequestWithValues(t, "POST", "/user/sign_up", map[string]string{
		"user_name":            "captcha-test",
		"email":                "captcha-test@example.com",
		"password":             "examplePassword!1",
		"retype":               "examplePassword!1",
		"img-captcha-id":       idCaptcha,
		"img-captcha-response": digitStr,
	})
	MakeRequest(t, req, http.StatusSeeOther)

	loginUserWithPassword(t, "captcha-test", "examplePassword!1")

	unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: "captcha-test", IsActive: true})
}

func TestSignupFormUI(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	t.Run("UI", func(t *testing.T) {
		// Mock alternative auth ways as enabled
		defer test.MockVariableValue(&setting.Service.EnableOpenIDSignIn, true)()
		defer test.MockVariableValue(&setting.Service.EnableOpenIDSignUp, true)()
		t.Run("Internal registration enabled", func(t *testing.T) {
			defer test.MockVariableValue(&setting.Service.AllowOnlyExternalRegistration, false)()
			req := NewRequest(t, "GET", "/user/sign_up")
			resp := MakeRequest(t, req, http.StatusOK)
			htmlDoc := NewHTMLParser(t, resp.Body)
			htmlDoc.AssertElement(t, "form[action='/user/sign_up'] input#user_name", true)
			htmlDoc.AssertElement(t, ".divider-text", true)
		})
		t.Run("Internal registration disabled", func(t *testing.T) {
			defer test.MockVariableValue(&setting.Service.AllowOnlyExternalRegistration, true)()
			req := NewRequest(t, "GET", "/user/sign_up")
			resp := MakeRequest(t, req, http.StatusOK)
			htmlDoc := NewHTMLParser(t, resp.Body)
			htmlDoc.AssertElement(t, "form[action='/user/sign_up'] input#user_name", false)
			htmlDoc.AssertElement(t, ".divider-text", false)
		})
	})
}
