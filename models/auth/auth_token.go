// Copyright 2023 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"forgejo.org/models/db"
	"forgejo.org/modules/timeutil"
	"forgejo.org/modules/util"
)

type AuthorizationPurpose string

var (
	// Used to store long term authorization tokens.
	LongTermAuthorization AuthorizationPurpose = "long_term_authorization"

	// Used to activate a user account.
	UserActivation AuthorizationPurpose = "user_activation"

	// Used to reset the password.
	PasswordReset AuthorizationPurpose = "password_reset"
)

// Used to activate the specified email address for a user.
func EmailActivation(email string) AuthorizationPurpose {
	return AuthorizationPurpose("email_activation:" + email)
}

// AuthorizationToken represents a authorization token to a user.
type AuthorizationToken struct {
	ID              int64  `xorm:"pk autoincr"`
	UID             int64  `xorm:"INDEX"`
	LookupKey       string `xorm:"INDEX UNIQUE"`
	HashedValidator string
	Purpose         AuthorizationPurpose `xorm:"NOT NULL DEFAULT 'long_term_authorization'"`
	Expiry          timeutil.TimeStamp
}

// TableName provides the real table name.
func (AuthorizationToken) TableName() string {
	return "forgejo_auth_token"
}

func init() {
	db.RegisterModel(new(AuthorizationToken))
}

// IsExpired returns if the authorization token is expired.
func (authToken *AuthorizationToken) IsExpired() bool {
	return authToken.Expiry.AsLocalTime().Before(time.Now())
}

// GenerateAuthToken generates a new authentication token for the given user.
// It returns the lookup key and validator values that should be passed to the
// user via a long-term cookie.
func GenerateAuthToken(ctx context.Context, userID int64, expiry timeutil.TimeStamp, purpose AuthorizationPurpose) (lookupKey, validator string, err error) {
	// Request 64 random bytes. The first 32 bytes will be used for the lookupKey
	// and the other 32 bytes will be used for the validator.
	rBytes := util.CryptoRandomBytes(64)
	hexEncoded := hex.EncodeToString(rBytes)
	validator, lookupKey = hexEncoded[64:], hexEncoded[:64]

	_, err = db.GetEngine(ctx).Insert(&AuthorizationToken{
		UID:             userID,
		Expiry:          expiry,
		LookupKey:       lookupKey,
		HashedValidator: HashValidator(rBytes[32:]),
		Purpose:         purpose,
	})
	return lookupKey, validator, err
}

// FindAuthToken will find a authorization token via the lookup key.
func FindAuthToken(ctx context.Context, lookupKey string, purpose AuthorizationPurpose) (*AuthorizationToken, error) {
	var authToken AuthorizationToken
	has, err := db.GetEngine(ctx).Where("lookup_key = ? AND purpose = ?", lookupKey, purpose).Get(&authToken)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, fmt.Errorf("lookup key %q: %w", lookupKey, util.ErrNotExist)
	}
	return &authToken, nil
}

// DeleteAuthToken will delete the authorization token.
func DeleteAuthToken(ctx context.Context, authToken *AuthorizationToken) error {
	_, err := db.DeleteByBean(ctx, authToken)
	return err
}

// DeleteAuthTokenByUser will delete all authorization tokens for the user.
func DeleteAuthTokenByUser(ctx context.Context, userID int64) error {
	if userID == 0 {
		return nil
	}

	_, err := db.DeleteByBean(ctx, &AuthorizationToken{UID: userID})
	return err
}

// HashValidator will return a hexified hashed version of the validator.
func HashValidator(validator []byte) string {
	h := sha256.New()
	h.Write(validator)
	return hex.EncodeToString(h.Sum(nil))
}
