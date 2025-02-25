// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user

import "code.gitea.io/gitea/modules/validation"

type FederatedUserFollower struct {
	ID int64 `xorm:"pk autoincr"`

	LocalUserID     int64 `xorm:"NOT NULL unique(fuf_rel)"`
	FederatedUserID int64 `xorm:"NOT NULL unique(fuf_rel)"`
}

func NewFederatedUserFollower(localUserID int64, federatedUserID int64) (FederatedUserFollower, error) {
	result := FederatedUserFollower{
		LocalUserID:     localUserID,
		FederatedUserID: federatedUserID,
	}
	if valid, err := validation.IsValid(result); !valid {
		return FederatedUserFollower{}, err
	}
	return result, nil
}

func (user FederatedUserFollower) Validate() []string {
	var result []string
	result = append(result, validation.ValidateNotEmpty(user.LocalUserID, "LocalUserID")...)
	result = append(result, validation.ValidateNotEmpty(user.FederatedUserID, "FederatedUserID")...)
	return result
}
