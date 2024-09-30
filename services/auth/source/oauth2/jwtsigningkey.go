// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package oauth2

import (
	"fmt"

	"code.gitea.io/gitea/modules/jwtx"
	"code.gitea.io/gitea/modules/setting"
)

// DefaultSigningKey is the default signing key for JWTs.
var DefaultSigningKey jwtx.JWTSigningKey

// InitSigningKey creates the default signing key from settings or creates a random key.
func InitSigningKey() error {
	var err error
	var key any

	switch setting.OAuth2.JWTSigningAlgorithm {
	case "HS256":
		fallthrough
	case "HS384":
		fallthrough
	case "HS512":
		key = setting.GetGeneralTokenSigningSecret()
	case "RS256":
		fallthrough
	case "RS384":
		fallthrough
	case "RS512":
		fallthrough
	case "ES256":
		fallthrough
	case "ES384":
		fallthrough
	case "ES512":
		fallthrough
	case "EdDSA":
		key, err = jwtx.LoadOrCreateAsymmetricKey(setting.OAuth2.JWTSigningPrivateKeyFile, setting.OAuth2.JWTSigningAlgorithm)
	default:
		return jwtx.ErrInvalidAlgorithmType{Algorithm: setting.OAuth2.JWTSigningAlgorithm}
	}

	if err != nil {
		return fmt.Errorf("Error while loading or creating JWT key: %w", err)
	}

	signingKey, err := jwtx.CreateJWTSigningKey(setting.OAuth2.JWTSigningAlgorithm, key)
	if err != nil {
		return err
	}

	DefaultSigningKey = signingKey

	return nil
}
