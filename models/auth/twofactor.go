// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package auth

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base32"
	"encoding/hex"
	"fmt"

	"forgejo.org/models/db"
	"forgejo.org/modules/keying"
	"forgejo.org/modules/timeutil"
	"forgejo.org/modules/util"

	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/pbkdf2"
)

//
// Two-factor authentication
//

// ErrTwoFactorNotEnrolled indicates that a user is not enrolled in two-factor authentication.
type ErrTwoFactorNotEnrolled struct {
	UID int64
}

// IsErrTwoFactorNotEnrolled checks if an error is a ErrTwoFactorNotEnrolled.
func IsErrTwoFactorNotEnrolled(err error) bool {
	_, ok := err.(ErrTwoFactorNotEnrolled)
	return ok
}

func (err ErrTwoFactorNotEnrolled) Error() string {
	return fmt.Sprintf("user not enrolled in 2FA [uid: %d]", err.UID)
}

// Unwrap unwraps this as a ErrNotExist err
func (err ErrTwoFactorNotEnrolled) Unwrap() error {
	return util.ErrNotExist
}

// TwoFactor represents a two-factor authentication token.
type TwoFactor struct {
	ID               int64  `xorm:"pk autoincr"`
	UID              int64  `xorm:"UNIQUE"`
	Secret           []byte `xorm:"BLOB"`
	ScratchSalt      string
	ScratchHash      string
	LastUsedPasscode string             `xorm:"VARCHAR(10)"`
	CreatedUnix      timeutil.TimeStamp `xorm:"INDEX created"`
	UpdatedUnix      timeutil.TimeStamp `xorm:"INDEX updated"`
}

func init() {
	db.RegisterModel(new(TwoFactor))
}

// GenerateScratchToken recreates the scratch token the user is using.
func (t *TwoFactor) GenerateScratchToken() string {
	// these chars are specially chosen, avoid ambiguous chars like `0`, `O`, `1`, `I`.
	const base32Chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	token := base32.NewEncoding(base32Chars).WithPadding(base32.NoPadding).EncodeToString(util.CryptoRandomBytes(6))
	t.ScratchSalt, _ = util.CryptoRandomString(10)
	t.ScratchHash = HashToken(token, t.ScratchSalt)
	return token
}

// HashToken return the hashable salt
func HashToken(token, salt string) string {
	tempHash := pbkdf2.Key([]byte(token), []byte(salt), 10000, 50, sha256.New)
	return hex.EncodeToString(tempHash)
}

// VerifyScratchToken verifies if the specified scratch token is valid.
func (t *TwoFactor) VerifyScratchToken(token string) bool {
	if len(token) == 0 {
		return false
	}
	tempHash := HashToken(token, t.ScratchSalt)
	return subtle.ConstantTimeCompare([]byte(t.ScratchHash), []byte(tempHash)) == 1
}

// SetSecret sets the 2FA secret.
func (t *TwoFactor) SetSecret(secretString string) {
	key := keying.DeriveKey(keying.ContextTOTP)
	t.Secret = key.Encrypt([]byte(secretString), keying.ColumnAndID("secret", t.ID))
}

// ValidateTOTP validates the provided passcode.
func (t *TwoFactor) ValidateTOTP(passcode string) (bool, error) {
	key := keying.DeriveKey(keying.ContextTOTP)
	secret, err := key.Decrypt(t.Secret, keying.ColumnAndID("secret", t.ID))
	if err != nil {
		return false, err
	}
	return totp.Validate(passcode, string(secret)), nil
}

// NewTwoFactor creates a new two-factor authentication token.
func NewTwoFactor(ctx context.Context, t *TwoFactor, secret string) error {
	return db.WithTx(ctx, func(ctx context.Context) error {
		sess := db.GetEngine(ctx)
		_, err := sess.Insert(t)
		if err != nil {
			return err
		}

		t.SetSecret(secret)
		_, err = sess.Cols("secret").ID(t.ID).Update(t)
		return err
	})
}

// UpdateTwoFactor updates a two-factor authentication token.
func UpdateTwoFactor(ctx context.Context, t *TwoFactor) error {
	_, err := db.GetEngine(ctx).ID(t.ID).AllCols().Update(t)
	return err
}

// GetTwoFactorByUID returns the two-factor authentication token associated with
// the user, if any.
func GetTwoFactorByUID(ctx context.Context, uid int64) (*TwoFactor, error) {
	twofa := &TwoFactor{}
	has, err := db.GetEngine(ctx).Where("uid=?", uid).Get(twofa)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, ErrTwoFactorNotEnrolled{uid}
	}
	return twofa, nil
}

// HasTwoFactorByUID returns the two-factor authentication token associated with
// the user, if any.
func HasTwoFactorByUID(ctx context.Context, uid int64) (bool, error) {
	return db.GetEngine(ctx).Where("uid=?", uid).Exist(&TwoFactor{})
}

// DeleteTwoFactorByID deletes two-factor authentication token by given ID.
func DeleteTwoFactorByID(ctx context.Context, id, userID int64) error {
	cnt, err := db.GetEngine(ctx).ID(id).Delete(&TwoFactor{
		UID: userID,
	})
	if err != nil {
		return err
	} else if cnt != 1 {
		return ErrTwoFactorNotEnrolled{userID}
	}
	return nil
}
