// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"testing"
	"time"

	auth_model "forgejo.org/models/auth"
	"forgejo.org/models/db"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/tests"

	"github.com/pquerna/otp/totp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPITwoFactor(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 16})

	req := NewRequest(t, "GET", "/api/v1/user").
		AddBasicAuth(user.Name)
	MakeRequest(t, req, http.StatusOK)

	otpKey, err := totp.Generate(totp.GenerateOpts{
		SecretSize:  40,
		Issuer:      "gitea-test",
		AccountName: user.Name,
	})
	require.NoError(t, err)

	tfa := &auth_model.TwoFactor{
		UID: user.ID,
	}

	require.NoError(t, auth_model.NewTwoFactor(db.DefaultContext, tfa, otpKey.Secret()))

	req = NewRequest(t, "GET", "/api/v1/user").
		AddBasicAuth(user.Name)
	MakeRequest(t, req, http.StatusUnauthorized)

	passcode, err := totp.GenerateCode(otpKey.Secret(), time.Now())
	require.NoError(t, err)

	req = NewRequest(t, "GET", "/api/v1/user").
		AddBasicAuth(user.Name)
	req.Header.Set("X-Gitea-OTP", passcode)
	MakeRequest(t, req, http.StatusOK)

	req = NewRequestf(t, "GET", "/api/v1/user").
		AddBasicAuth(user.Name)
	req.Header.Set("X-Forgejo-OTP", passcode)
	MakeRequest(t, req, http.StatusOK)
}

func TestAPIWebAuthn(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 32})
	unittest.AssertExistsAndLoadBean(t, &auth_model.WebAuthnCredential{UserID: user.ID})

	req := NewRequest(t, "GET", "/api/v1/user")
	req.SetBasicAuth(user.Name, "notpassword")

	resp := MakeRequest(t, req, http.StatusUnauthorized)

	type userResponse struct {
		Message string `json:"message"`
	}
	var userParsed userResponse

	DecodeJSON(t, resp, &userParsed)

	assert.Equal(t, "Basic authorization is not allowed while having security keys enrolled", userParsed.Message)
}
