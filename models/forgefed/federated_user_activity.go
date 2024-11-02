// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgefed

import (
	"context"
	"fmt"

	"code.gitea.io/gitea/models/db"
	user_model "code.gitea.io/gitea/models/user"
	fm "code.gitea.io/gitea/modules/forgefed"
	"code.gitea.io/gitea/modules/json"
	"code.gitea.io/gitea/modules/timeutil"
)

type FederatedUserActivity struct {
	ID     int64 `xorm:"pk autoincr"`
	UserID int64 `xorm:"NOT NULL"`

	ExternalID  string           `xorm:"NOT NULL"`
	Actor       *user_model.User `xorm:"-"`
	Note        string
	OriginalURL string

	Original string

	Created timeutil.TimeStamp `xorm:"created"`
}

type FederatedUserFollower struct {
	ID int64 `xorm:"pk autoincr"`

	LocalUserID     int64 `xorm:"NOT NULL unique(fuf_rel)"`
	FederatedUserID int64 `xorm:"NOT NULL unique(fuf_rel)"`
}

func init() {
	db.RegisterModel(new(FederatedUserActivity))
	db.RegisterModel(new(FederatedUserFollower))
}

func (fua *FederatedUserActivity) LoadActor(ctx context.Context) error {
	if fua.Actor != nil {
		return nil
	}

	actor, err := user_model.GetUserByActorURL(ctx, fua.ExternalID)
	if err != nil {
		return err
	}
	fua.Actor = actor

	return nil
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

func AddUserActivity(ctx context.Context, userID int64, externalID string, activity *fm.ForgeUserActivityNote) error {
	json, err := json.Marshal(activity)
	if err != nil {
		return err
	}
	_, err = db.GetEngine(ctx).
		Insert(&FederatedUserActivity{
			UserID:      userID,
			ExternalID:  externalID,
			Note:        activity.Content.String(),
			OriginalURL: activity.URL.GetID().String(),
			Original:    string(json),
		})
	return err
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

type GetFollowingFeedsOptions struct {
	db.ListOptions
	Actor *user_model.User
}

func GetFollowingFeeds(ctx context.Context, opts GetFollowingFeedsOptions) ([]*FederatedUserActivity, int64, error) {
	sess := db.GetEngine(ctx).Where("user_id = ?", opts.Actor.ID)
	opts.SetDefaultValues()
	sess = db.SetSessionPagination(sess, &opts)

	actions := make([]*FederatedUserActivity, 0, opts.PageSize)
	count, err := sess.FindAndCount(&actions)
	if err != nil {
		return nil, 0, fmt.Errorf("FindAndCount: %w", err)
	}
	for _, act := range actions {
		if err := act.LoadActor(ctx); err != nil {
			return nil, 0, err
		}
	}
	return actions, count, err
}
