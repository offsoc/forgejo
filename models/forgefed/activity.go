// Copyright 2023 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgefed

import (
	"time"

	"code.gitea.io/gitea/modules/validation"

	ap "github.com/go-ap/activitypub"
)

// ForgeLike activity data type
// swagger:model
type ForgeLike struct {
	// swagger:ignore
	ap.Activity
}

func NewForgeLike(actorIRI, objectIRI string, startTime time.Time) (ForgeLike, error) {
	result := ForgeLike{}
	result.Type = ap.LikeType
	// ToDo: Would validating the source by Actor.Type field make sense?
	result.Actor = ap.IRI(actorIRI)   // Thats us, a User
	result.Object = ap.IRI(objectIRI) // Thats them, a Repository
	result.StartTime = startTime
	if valid, err := validation.IsValid(result); !valid {
		return ForgeLike{}, err
	}
	return result, nil
}

func (like ForgeLike) MarshalJSON() ([]byte, error) {
	return like.Activity.MarshalJSON()
}

func (like *ForgeLike) UnmarshalJSON(data []byte) error {
	return like.Activity.UnmarshalJSON(data)
}

func (like ForgeLike) IsNewer(compareTo time.Time) bool {
	return like.StartTime.After(compareTo)
}

func (like ForgeLike) Validate() []string {
	var result []string
	result = append(result, validation.ValidateNotEmpty(string(like.Type), "type")...)
	result = append(result, validation.ValidateOneOf(string(like.Type), []any{"Like"})...)
	result = append(result, validation.ValidateNotEmpty(like.Actor.GetID().String(), "actor")...)
	result = append(result, validation.ValidateNotEmpty(like.Object.GetID().String(), "object")...)
	result = append(result, validation.ValidateNotEmpty(like.StartTime.String(), "startTime")...)
	if like.StartTime.IsZero() {
		result = append(result, "StartTime was invalid.")
	}

	return result
}
