// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user

import (
	"context"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/modules/validation"
)

type FederatedUser struct {
	ID               int64  `xorm:"pk autoincr"`
	UserID           int64  `xorm:"NOT NULL"`
	ExternalID       string `xorm:"UNIQUE(federation_user_mapping) NOT NULL"`
	FederationHostID int64  `xorm:"UNIQUE(federation_user_mapping) NOT NULL"`

	InboxURL *string
	ActorURL *string
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

func (user FederatedUser) Validate() []string {
	var result []string
	result = append(result, validation.ValidateNotEmpty(user.UserID, "UserID")...)
	result = append(result, validation.ValidateNotEmpty(user.ExternalID, "ExternalID")...)
	result = append(result, validation.ValidateNotEmpty(user.FederationHostID, "FederationHostID")...)
	return result
}

func (user *FederatedUser) SetInboxURL(ctx context.Context, url *string) error {
	user.InboxURL = url
	_, err := db.GetEngine(ctx).ID(user.ID).Cols("inbox_url").Update(user)
	return err
}

func GetFederatedUserByID(ctx context.Context, id int64) (*FederatedUser, error) {
	var user FederatedUser
	_, err := db.GetEngine(ctx).Where("id = ?", id).Get(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func GetUserByActorURL(ctx context.Context, actorURL string) (*User, error) {
	var user User
	_, err := db.GetEngine(ctx).Table("`user`").Join("INNER", "`federated_user`", "`user`.id = `federated_user`.user_id").Where("`federated_user`.actor_url = ?", actorURL).Get(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}
