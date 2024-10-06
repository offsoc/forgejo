// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"testing"
	"time"

	auth_model "code.gitea.io/gitea/models/auth"
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/tests"

	"github.com/pquerna/otp/totp"
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
	require.NoError(t, tfa.SetSecret(otpKey.Secret()))

	require.NoError(t, auth_model.NewTwoFactor(db.DefaultContext, tfa))

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
