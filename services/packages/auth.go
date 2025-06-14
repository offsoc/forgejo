// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package packages

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	auth_model "forgejo.org/models/auth"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"

	"github.com/golang-jwt/jwt/v5"
)

type packageClaims struct {
	jwt.RegisteredClaims
	UserID int64
	Scope  auth_model.AccessTokenScope
}

func CreateAuthorizationToken(u *user_model.User, scope auth_model.AccessTokenScope) (string, error) {
	now := time.Now()

	claims := packageClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(24 * time.Hour)),
			NotBefore: jwt.NewNumericDate(now),
		},
		UserID: u.ID,
		Scope:  scope,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString(setting.GetGeneralTokenSigningSecret())
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func ParseAuthorizationToken(req *http.Request) (int64, auth_model.AccessTokenScope, error) {
	h := req.Header.Get("Authorization")
	if h == "" {
		return 0, "", nil
	}

	parts := strings.SplitN(h, " ", 2)
	if len(parts) != 2 {
		log.Error("split token failed: %s", h)
		return 0, "", errors.New("split token failed")
	}

	token, err := jwt.ParseWithClaims(parts[1], &packageClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return setting.GetGeneralTokenSigningSecret(), nil
	})
	if err != nil {
		return 0, "", err
	}

	c, ok := token.Claims.(*packageClaims)
	if !token.Valid || !ok {
		return 0, "", errors.New("invalid token claim")
	}

	return c.UserID, c.Scope, nil
}
