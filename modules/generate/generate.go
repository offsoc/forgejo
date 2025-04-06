// Copyright 2016 The Gogs Authors. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package generate

import (
	"encoding/base64"
	"fmt"
	"time"

	"forgejo.org/modules/util"

	"github.com/golang-jwt/jwt/v5"
)

// NewInternalToken generate a new value intended to be used by INTERNAL_TOKEN.
func NewInternalToken() (string, error) {
	secretKey := base64.RawURLEncoding.EncodeToString(util.CryptoRandomBytes(32))

	now := time.Now()

	internalToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"nbf": now.Unix(),
	}).SignedString([]byte(secretKey))
	if err != nil {
		return "", err
	}

	return internalToken, nil
}

const defaultJwtSecretLen = 32

// DecodeJwtSecret decodes a base64 encoded jwt secret into bytes, and check its length
func DecodeJwtSecret(src string) ([]byte, error) {
	encoding := base64.RawURLEncoding
	decoded := make([]byte, encoding.DecodedLen(len(src))+3)
	if n, err := encoding.Decode(decoded, []byte(src)); err != nil {
		return nil, err
	} else if n != defaultJwtSecretLen {
		return nil, fmt.Errorf("invalid base64 decoded length: %d, expects: %d", n, defaultJwtSecretLen)
	}
	return decoded[:defaultJwtSecretLen], nil
}

// NewJwtSecret generates a new base64 encoded value intended to be used for JWT secrets.
func NewJwtSecret() ([]byte, string) {
	bytes := util.CryptoRandomBytes(32)
	return bytes, base64.RawURLEncoding.EncodeToString(bytes)
}

// NewSecretKey generate a new value intended to be used by SECRET_KEY.
func NewSecretKey() (string, error) {
	secretKey, err := util.CryptoRandomString(64)
	if err != nil {
		return "", err
	}

	return secretKey, nil
}
