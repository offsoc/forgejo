// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user

import "forgejo.org/modules/validation"

type FederatedUserFollower struct {
	ID              int64 `xorm:"pk autoincr"`
	FollowedUserID  int64 `xorm:"NOT NULL unique(fuf_rel)"`
	FollowingUserID int64 `xorm:"NOT NULL unique(fuf_rel)"`
}

func NewFederatedUserFollower(followedUserID, federatedUserID int64) (FederatedUserFollower, error) {
	result := FederatedUserFollower{
		FollowedUserID:  followedUserID,
		FollowingUserID: federatedUserID,
	}
	if valid, err := validation.IsValid(result); !valid {
		return FederatedUserFollower{}, err
	}
	return result, nil
}

func (user FederatedUserFollower) Validate() []string {
	var result []string
	result = append(result, validation.ValidateNotEmpty(user.FollowedUserID, "FollowedUserID")...)
	result = append(result, validation.ValidateNotEmpty(user.FollowingUserID, "FollowingUserID")...)
	return result
}
