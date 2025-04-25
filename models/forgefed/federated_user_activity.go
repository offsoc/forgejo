// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

// TODO: create package by its own
package forgefed

import (
	"context"
	"fmt"

	"forgejo.org/models/db"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/json"
	"forgejo.org/modules/timeutil"
	"forgejo.org/modules/validation"
	ap "github.com/go-ap/activitypub"
)

type FederatedUserActivity struct {
	ID           int64 `xorm:"pk autoincr"`
	UserID       int64 `xorm:"NOT NULL"`
	ActorID      string
	Actor        *user_model.User `xorm:"-"` // transient
	NoteContent  string
	NoteURL      string
	OriginalNote string
	Created      timeutil.TimeStamp `xorm:"created"`
}

func init() {
	db.RegisterModel(new(FederatedUserActivity))
}

func NewFederatedUserActivity(userID int64, actorID, noteContent, noteURL string, originalNote ap.Activity) (FederatedUserActivity, error) {
	json, err := json.Marshal(originalNote)
	if err != nil {
		return FederatedUserActivity{}, err
	}
	result := FederatedUserActivity{
		UserID:       userID,
		ActorID:      actorID,
		NoteContent:  noteContent,
		NoteURL:      noteURL,
		OriginalNote: string(json),
	}
	if valid, err := validation.IsValid(result); !valid {
		return FederatedUserActivity{}, err
	}
	return result, nil
}

// TODO: add tests
func (federatedUser FederatedUserActivity) Validate() []string {
	var result []string
	result = append(result, validation.ValidateNotEmpty(federatedUser.UserID, "UserID")...)
	result = append(result, validation.ValidateNotEmpty(federatedUser.ActorID, "ActorID")...)
	result = append(result, validation.ValidateNotEmpty(federatedUser.NoteContent, "NoteContent")...)
	result = append(result, validation.ValidateNotEmpty(federatedUser.NoteURL, "NoteURL")...)
	result = append(result, validation.ValidateNotEmpty(federatedUser.OriginalNote, "OriginalNote")...)
	return result
}

func CreateUserActivity(ctx context.Context, federatedUserActivity *FederatedUserActivity) error {
	if valid, err := validation.IsValid(federatedUserActivity); !valid {
		return err
	}
	_, err := db.GetEngine(ctx).Insert(federatedUserActivity)
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
		if err := act.loadActor(ctx); err != nil {
			return nil, 0, err
		}
	}
	return actions, count, err
}

// TODO: move this to service as the operation crosses the aggregate borders
func (fua *FederatedUserActivity) loadActor(ctx context.Context) error {
	actorUser, _, err := user_model.GetFederatedUserByUserId(ctx, fua.UserID)
	if err != nil {
		return err
	}
	fua.Actor = actorUser

	return nil
}
