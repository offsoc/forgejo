// Copyright 2024, 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgefed

import (
	"forgejo.org/models/user"

	"forgejo.org/modules/validation"

	ap "github.com/go-ap/activitypub"
	"github.com/google/uuid"
)

type ForgeFollow struct {
	// swagger:ignore
	ap.Activity
}

func NewForgeFollow(localUser *user.User, actorURI string) (ForgeFollow, error) {
	result := ForgeFollow{}
	result.Activity = *ap.FollowNew(
		ap.IRI(localUser.APActorID()+"/follows/"+uuid.New().String()),
		ap.IRI(actorURI),
	)
	result.Actor = ap.IRI(localUser.APActorID())
	result.Target = ap.IRI(actorURI)

	if valid, err := validation.IsValid(result); !valid {
		return ForgeFollow{}, err
	}

	return result, nil
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
