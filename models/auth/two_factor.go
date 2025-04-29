// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT
package auth

import (
	"context"
)

// HasTwoFactorByUID returns true if the user has TOTP or WebAuthn enabled for
// their account.
func HasTwoFactorByUID(ctx context.Context, userID int64) (bool, error) {
	hasTOTP, err := HasTOTPByUID(ctx, userID)
	if err != nil {
		return false, err
	}
	if hasTOTP {
		return true, nil
	}

	return HasWebAuthnRegistrationsByUID(ctx, userID)
}
