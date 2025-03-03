// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user

import (
	"context"
	"database/sql"

	"forgejo.org/models/db"
	"forgejo.org/modules/validation"
)

type FederatedUser struct {
	ID               int64                  `xorm:"pk autoincr"`
	UserID           int64                  `xorm:"NOT NULL"`
	ExternalID       string                 `xorm:"UNIQUE(federation_user_mapping) NOT NULL"`
	FederationHostID int64                  `xorm:"UNIQUE(federation_user_mapping) NOT NULL"`
	KeyID            sql.NullString         `xorm:"key_id UNIQUE"`
	PublicKey        sql.Null[sql.RawBytes] `xorm:"BLOB"`
}

func NewFederatedUser(userID int64, externalID string, federationHostID int64) (FederatedUser, error) {
	result := FederatedUser{
		UserID:           userID,
		ExternalID:       externalID,
		FederationHostID: federationHostID,
	}
	if valid, err := validation.IsValid(result); !valid {
		return FederatedUser{}, err
	}
	return result, nil
}

func getFederatedUserFromDB(ctx context.Context, searchKey, searchValue any) (*FederatedUser, error) {
	federatedUser := new(FederatedUser)
	has, err := db.GetEngine(ctx).Where(searchKey, searchValue).Get(federatedUser)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, nil
	}

	if res, err := validation.IsValid(*federatedUser); !res {
		return nil, err
	}

	return federatedUser, nil
}

func GetFederatedUserByKeyID(ctx context.Context, keyID string) (*FederatedUser, error) {
	return getFederatedUserFromDB(ctx, "key_id=?", keyID)
}

func GetFederatedUserByUserID(ctx context.Context, userID int64) (*FederatedUser, error) {
	return getFederatedUserFromDB(ctx, "user_id=?", userID)
}

func (user FederatedUser) Validate() []string {
	var result []string
	result = append(result, validation.ValidateNotEmpty(user.UserID, "UserID")...)
	result = append(result, validation.ValidateNotEmpty(user.ExternalID, "ExternalID")...)
	result = append(result, validation.ValidateNotEmpty(user.FederationHostID, "FederationHostID")...)
	return result
}
