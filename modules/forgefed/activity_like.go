// Copyright 2023, 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgefed

import (
	"time"

	"forgejo.org/modules/validation"

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
	result.Actor = ap.IRI(actorIRI)
	result.Object = ap.IRI(objectIRI)
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
	result = append(result, validation.ValidateOneOf(string(like.Type), []any{"Like"}, "type")...)

	if like.Actor == nil {
		result = append(result, "Actor should not be nil.")
	} else {
		result = append(result, validation.ValidateNotEmpty(like.Actor.GetID().String(), "actor")...)
	}

	result = append(result, validation.ValidateNotEmpty(like.StartTime.String(), "startTime")...)
	if like.StartTime.IsZero() {
		result = append(result, "StartTime was invalid.")
	}

	if like.Object == nil {
		result = append(result, "Object should not be nil.")
	} else {
		result = append(result, validation.ValidateNotEmpty(like.Object.GetID().String(), "object")...)
	}

	return result
}
