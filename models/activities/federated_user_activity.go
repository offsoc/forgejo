// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package activities

import (
	"context"
	"fmt"

	"forgejo.org/models/db"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/json"
	"forgejo.org/modules/log"
	"forgejo.org/modules/timeutil"
	"forgejo.org/modules/validation"

	ap "github.com/go-ap/activitypub"
)

type FederatedUserActivity struct {
	ID           int64 `xorm:"pk autoincr"`
	UserID       int64 `xorm:"NOT NULL"`
	ActorID      int64
	ActorURI     string
	Actor        *user_model.User   `xorm:"-"` // transient
	NoteContent  string             `xorm:"TEXT"`
	NoteURL      string             `xorm:"VARCHAR(255)"`
	OriginalNote string             `xorm:"TEXT"`
	Created      timeutil.TimeStamp `xorm:"created"`
}

func init() {
	db.RegisterModel(new(FederatedUserActivity))
}

func NewFederatedUserActivity(userID, actorID int64, actorURI, noteContent, noteURL string, originalNote ap.Activity) (FederatedUserActivity, error) {
	jsonString, err := json.Marshal(originalNote)
	if err != nil {
		return FederatedUserActivity{}, err
	}
	result := FederatedUserActivity{
		UserID:       userID,
		ActorID:      actorID,
		ActorURI:     actorURI,
		NoteContent:  noteContent,
		NoteURL:      noteURL,
		OriginalNote: string(jsonString),
	}
	if valid, err := validation.IsValid(result); !valid {
		return FederatedUserActivity{}, err
	}
	return result, nil
}

func (federatedUserActivity FederatedUserActivity) Validate() []string {
	var result []string
	result = append(result, validation.ValidateNotEmpty(federatedUserActivity.UserID, "UserID")...)
	result = append(result, validation.ValidateNotEmpty(federatedUserActivity.ActorID, "ActorID")...)
	result = append(result, validation.ValidateNotEmpty(federatedUserActivity.ActorURI, "ActorURI")...)
	result = append(result, validation.ValidateNotEmpty(federatedUserActivity.NoteContent, "NoteContent")...)
	result = append(result, validation.ValidateNotEmpty(federatedUserActivity.NoteURL, "NoteURL")...)
	result = append(result, validation.ValidateNotEmpty(federatedUserActivity.OriginalNote, "OriginalNote")...)
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
}

func GetFollowingFeeds(ctx context.Context, actorID int64, opts GetFollowingFeedsOptions) ([]*FederatedUserActivity, int64, error) {
	log.Debug("user_id = %s", actorID)
	sess := db.GetEngine(ctx).Where("user_id = ?", actorID)
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

func (federatedUserActivity *FederatedUserActivity) loadActor(ctx context.Context) error {
	log.Debug("for activity %s", federatedUserActivity)
	actorUser, _, err := user_model.GetFederatedUserByUserID(ctx, federatedUserActivity.ActorID)
	if err != nil {
		return err
	}
	federatedUserActivity.Actor = actorUser

	return nil
}
