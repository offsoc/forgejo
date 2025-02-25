// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user

import (
	"context"
	"fmt"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/modules/optional"
	"code.gitea.io/gitea/modules/validation"
)

func init() {
	db.RegisterModel(new(FederatedUser))
	db.RegisterModel(new(FederatedUserFollower))
}

func CreateFederatedUser(ctx context.Context, user *User, federatedUser *FederatedUser) error {
	if res, err := validation.IsValid(user); !res {
		return err
	}
	overwrite := CreateUserOverwriteOptions{
		IsActive:     optional.Some(false),
		IsRestricted: optional.Some(false),
	}

	// Begin transaction
	ctx, committer, err := db.TxContext((ctx))
	if err != nil {
		return err
	}
	defer committer.Close()

	if err := CreateUser(ctx, user, &overwrite); err != nil {
		return err
	}

	federatedUser.UserID = user.ID
	if res, err := validation.IsValid(federatedUser); !res {
		return err
	}

	_, err = db.GetEngine(ctx).Insert(federatedUser)
	if err != nil {
		return err
	}

	// Commit transaction
	return committer.Commit()
}

func FindFederatedUser(ctx context.Context, externalID string,
	federationHostID int64,
) (*User, *FederatedUser, error) {
	federatedUser := new(FederatedUser)
	user := new(User)
	has, err := db.GetEngine(ctx).Where("external_id=? and federation_host_id=?", externalID, federationHostID).Get(federatedUser)
	if err != nil {
		return nil, nil, err
	} else if !has {
		return nil, nil, nil
	}
	has, err = db.GetEngine(ctx).ID(federatedUser.UserID).Get(user)
	if err != nil {
		return nil, nil, err
	} else if !has {
		return nil, nil, fmt.Errorf("User %v for federated user is missing", federatedUser.UserID)
	}

	if res, err := validation.IsValid(*user); !res {
		return nil, nil, err
	}
	if res, err := validation.IsValid(*federatedUser); !res {
		return nil, nil, err
	}
	return user, federatedUser, nil
}

func DeleteFederatedUser(ctx context.Context, userID int64) error {
	_, err := db.GetEngine(ctx).Delete(&FederatedUser{UserID: userID})
	return err
}

func GetFollowersForUserID(ctx context.Context, userID int64) ([]*FederatedUserFollower, error) {
	followers := make([]*FederatedUserFollower, 0, 8)

	err := db.GetEngine(ctx).
		Where("local_user_id = ?", userID).
		Find(&followers)
	if err != nil {
		return nil, err
	}
	return followers, nil
}

func AddFollower(ctx context.Context, followedUserID, followingUserID int64) (int64, error) {
	followerID, err := db.GetEngine(ctx).
		Insert(&FederatedUserFollower{
			LocalUserID:     followedUserID,
			FederatedUserID: followingUserID,
		})
	return followerID, err
}

func RemoveFollower(ctx context.Context, localUserID, federatedUserID int64) error {
	_, err := db.GetEngine(ctx).Delete(&FederatedUserFollower{
		LocalUserID:     localUserID,
		FederatedUserID: federatedUserID,
	})
	return err
}

func IsFollowingAp(ctx context.Context, followedUserID, followingUserID int64) (bool, error) {
	return db.GetEngine(ctx).Get(&FederatedUserFollower{
		LocalUserID:     followedUserID,
		FederatedUserID: followingUserID,
	})
}