// Copyright 2024, 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user

import (
	"context"
	"fmt"

	"forgejo.org/models/db"
	"forgejo.org/modules/log"
	"forgejo.org/modules/optional"
	"forgejo.org/modules/validation"
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
	defer func(committer db.Committer) {
		err := committer.Close()
		if err != nil {
			log.Error("Error closing committer: %v", err)
		}
	}(committer)

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

func (federatedUser *FederatedUser) UpdateFederatedUser(ctx context.Context) error {
	if _, err := validation.IsValid(federatedUser); err != nil {
		return err
	}
	_, err := db.GetEngine(ctx).ID(federatedUser.ID).Cols("inbox_path").Update(federatedUser)
	return err
}

func FindFederatedUser(ctx context.Context, externalID string, federationHostID int64) (*User, *FederatedUser, error) {
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

func GetFederatedUser(ctx context.Context, externalID string, federationHostID int64) (*User, *FederatedUser, error) {
	user, federatedUser, err := FindFederatedUser(ctx, externalID, federationHostID)
	if err != nil {
		return nil, nil, err
	} else if federatedUser == nil {
		return nil, nil, fmt.Errorf("FederatedUser for externalId = %v and federationHostId = %v does not exist", externalID, federationHostID)
	}
	return user, federatedUser, nil
}

func GetFederatedUserByUserID(ctx context.Context, userID int64) (*User, *FederatedUser, error) {
	federatedUser := new(FederatedUser)
	user := new(User)
	has, err := db.GetEngine(ctx).Where("user_id=?", userID).Get(federatedUser)
	if err != nil {
		return nil, nil, err
	} else if !has {
		return nil, nil, fmt.Errorf("Federated user %v does not exist", federatedUser.UserID)
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

func FindFederatedUserByKeyID(ctx context.Context, keyID string) (*User, *FederatedUser, error) {
	federatedUser := new(FederatedUser)
	user := new(User)
	has, err := db.GetEngine(ctx).Where("key_id=?", keyID).Get(federatedUser)
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

func UpdateFederatedUser(ctx context.Context, fedearatedUser *FederatedUser) error {
	if res, err := validation.IsValid(fedearatedUser); !res {
		return err
	}
	_, err := db.GetEngine(ctx).ID(fedearatedUser.ID).Update(fedearatedUser)
	return err
}

func DeleteFederatedUser(ctx context.Context, userID int64) error {
	_, err := db.GetEngine(ctx).Delete(&FederatedUser{UserID: userID})
	return err
}

func GetFollowersForUser(ctx context.Context, user *User) ([]*FederatedUserFollower, error) {
	if res, err := validation.IsValid(user); !res {
		return nil, err
	}
	followers := make([]*FederatedUserFollower, 0, 8)

	err := db.GetEngine(ctx).
		Where("followed_user_id = ?", user.ID).
		Find(&followers)
	if err != nil {
		return nil, err
	}
	for _, element := range followers {
		if res, err := validation.IsValid(*element); !res {
			return nil, err
		}
	}
	return followers, nil
}

func AddFollower(ctx context.Context, followedUser *User, followingUser *FederatedUser) (*FederatedUserFollower, error) {
	if res, err := validation.IsValid(followedUser); !res {
		return nil, err
	}
	if res, err := validation.IsValid(followingUser); !res {
		return nil, err
	}

	federatedUserFollower, err := NewFederatedUserFollower(followedUser.ID, followingUser.UserID)
	if err != nil {
		return nil, err
	}
	_, err = db.GetEngine(ctx).Insert(&federatedUserFollower)
	if err != nil {
		return nil, err
	}

	return &federatedUserFollower, err
}

func RemoveFollower(ctx context.Context, followedUser *User, followingUser *FederatedUser) error {
	if res, err := validation.IsValid(followedUser); !res {
		return err
	}
	if res, err := validation.IsValid(followingUser); !res {
		return err
	}

	_, err := db.GetEngine(ctx).Delete(&FederatedUserFollower{
		FollowedUserID:  followedUser.ID,
		FollowingUserID: followingUser.UserID,
	})
	return err
}

// TODO: We should unify Activity-pub-following and classical following (see models/user/follow.go)
func IsFollowingAp(ctx context.Context, followedUser *User, followingUser *FederatedUser) (bool, error) {
	if res, err := validation.IsValid(followedUser); !res {
		return false, err
	}
	if res, err := validation.IsValid(followingUser); !res {
		return false, err
	}

	return db.GetEngine(ctx).Get(&FederatedUserFollower{
		FollowedUserID:  followedUser.ID,
		FollowingUserID: followingUser.UserID,
	})
}
