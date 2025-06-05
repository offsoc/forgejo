// Copyright 2024, 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user

import (
	"database/sql"

	"forgejo.org/modules/validation"
)

type FederatedUser struct {
	ID                    int64                  `xorm:"pk autoincr"`
	UserID                int64                  `xorm:"NOT NULL INDEX user_id"`
	ExternalID            string                 `xorm:"UNIQUE(federation_user_mapping) NOT NULL"`
	FederationHostID      int64                  `xorm:"UNIQUE(federation_user_mapping) NOT NULL"`
	KeyID                 sql.NullString         `xorm:"key_id UNIQUE"`
	PublicKey             sql.Null[sql.RawBytes] `xorm:"BLOB"`
	InboxPath             string
	NormalizedOriginalURL string // This field is just to keep original information. Pls. do not use for search or as ID!
}

func NewFederatedUser(userID int64, externalID string, federationHostID int64, inboxPath, normalizedOriginalURL string) (FederatedUser, error) {
	result := FederatedUser{
		UserID:                userID,
		ExternalID:            externalID,
		FederationHostID:      federationHostID,
		InboxPath:             inboxPath,
		NormalizedOriginalURL: normalizedOriginalURL,
	}
	if valid, err := validation.IsValid(result); !valid {
		return FederatedUser{}, err
	}
	return result, nil
}

func (federatedUser FederatedUser) Validate() []string {
	var result []string
	result = append(result, validation.ValidateNotEmpty(federatedUser.UserID, "UserID")...)
	result = append(result, validation.ValidateNotEmpty(federatedUser.ExternalID, "ExternalID")...)
	result = append(result, validation.ValidateNotEmpty(federatedUser.FederationHostID, "FederationHostID")...)
	result = append(result, validation.ValidateNotEmpty(federatedUser.InboxPath, "InboxPath")...)
	return result
}
