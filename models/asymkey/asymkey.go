// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later
package asymkey

import (
	"context"

	"forgejo.org/models/db"
)

// HasAsymKeyByUID returns true if the user has a GPG key or SSH key associated
// with its account.
func HasAsymKeyByUID(ctx context.Context, userID int64) (bool, error) {
	hasGPGKey, err := db.Exist[GPGKey](ctx, FindGPGKeyOptions{
		OwnerID:        userID,
		IncludeSubKeys: true,
	}.ToConds())
	if err != nil {
		return false, err
	}
	if hasGPGKey {
		return true, nil
	}

	return db.Exist[PublicKey](ctx, FindPublicKeyOptions{
		OwnerID:  userID,
		KeyTypes: []KeyType{KeyTypeUser},
	}.ToConds())
}
