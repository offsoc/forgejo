// Copyright 2024, 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgefed

import (
	"forgejo.org/modules/validation"

	ap "github.com/go-ap/activitypub"
	"github.com/google/uuid"
)

// ForgeLike activity data type
// swagger:model
type ForgeFollow struct {
	// swagger:ignore
	ap.Activity
}

func NewForgeFollowFromActivity(activity ap.Activity) (ForgeFollow, error) {
	result := ForgeFollow{}
	result.Activity = activity
	if valid, err := validation.IsValid(result); !valid {
		return ForgeFollow{}, err
	}
	return result, nil
}

func NewForgeFollow(actor, object string) (ForgeFollow, error) {
	result := ForgeFollow{}
	result.Type = ap.FollowType
	result.ID = ap.IRI(actor + "/follows/" + uuid.New().String())
	result.Actor = ap.IRI(actor)
	result.Object = ap.IRI(object)
	if valid, err := validation.IsValid(result); !valid {
		return ForgeFollow{}, err
	}
	return result, nil
}

func (follow ForgeFollow) MarshalJSON() ([]byte, error) {
	return follow.Activity.MarshalJSON()
}

func (follow *ForgeFollow) UnmarshalJSON(data []byte) error {
	return follow.Activity.UnmarshalJSON(data)
}

func (follow ForgeFollow) Validate() []string {
	var result []string
	if follow.Actor == nil {
		result = append(result, "Actor should not be nil.")
	} else {
		result = append(result, validation.ValidateNotEmpty(string(follow.Type), "type")...)
		result = append(result, validation.ValidateNotEmpty(follow.Actor.GetID().String(), "actor")...)
		result = append(result, validation.ValidateNotEmpty(follow.Object.GetID().String(), "object")...)
	}

	return result
}
