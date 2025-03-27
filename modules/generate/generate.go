// Copyright 2016 The Gogs Authors. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package generate

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"time"

	"forgejo.org/modules/util"

	"github.com/golang-jwt/jwt/v5"
)

// NewInternalToken generate a new value intended to be used by INTERNAL_TOKEN.
func NewInternalToken() (string, error) {
	secretBytes := make([]byte, 32)
	_, err := io.ReadFull(rand.Reader, secretBytes)
	if err != nil {
		return "", err
	}

	secretKey := base64.RawURLEncoding.EncodeToString(secretBytes)

	now := time.Now()

	var internalToken string
	internalToken, err = jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
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
func NewJwtSecret() ([]byte, string, error) {
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		return nil, "", err
	}

	return bytes, base64.RawURLEncoding.EncodeToString(bytes), nil
}

// NewSecretKey generate a new value intended to be used by SECRET_KEY.
func NewSecretKey() (string, error) {
	secretKey, err := util.CryptoRandomString(64)
	if err != nil {
		return "", err
	}

	return secretKey, nil
}
