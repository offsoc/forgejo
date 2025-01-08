// Copyright 2023, 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgefed

import (
	"time"

	"code.gitea.io/gitea/modules/validation"

	ap "github.com/go-ap/activitypub"
)

type ForgeUndoLike struct {
	// swagger:ignore
	ap.Activity
}

func NewForgeUndoLike(actorIRI, objectIRI string, startTime time.Time) (ForgeUndoLike, error) {
	result := ForgeUndoLike{}
	result.Type = ap.UndoType
	result.Actor = ap.IRI(actorIRI) // User who triggered the UndoLike (must be same as User who triggered the initial Like)
	result.StartTime = startTime

	like := ap.Activity{} // The Like, which should be undone (similar to struct Like, but without start date)
	like.Type = ap.LikeType
	like.Actor = ap.IRI(actorIRI)   // User of the Like which should be undone
	like.Object = ap.IRI(objectIRI) // Repository of the Like which should be undone
	result.Object = &like

	if valid, err := validation.IsValid(result); !valid {
		return ForgeUndoLike{}, err
	}
	return result, nil
}

func (undo *ForgeUndoLike) UnmarshalJSON(data []byte) error {
	return undo.Activity.UnmarshalJSON(data)
}
	
func (undo ForgeUndoLike) Validate() []string {
		var result []string
		result = append(result, validation.ValidateNotEmpty(string(undo.Type), "type")...)
		result = append(result, validation.ValidateOneOf(string(undo.Type), []any{"Undo"}, "type")...)
	
		if undo.Actor == nil {
			result = append(result, "Actor should not be nil.")
		} else {
			result = append(result, validation.ValidateNotEmpty(undo.Actor.GetID().String(), "actor")...)
		}
	
		result = append(result, validation.ValidateNotEmpty(undo.StartTime.String(), "startTime")...)
		if undo.StartTime.IsZero() {
			result = append(result, "StartTime was invalid.")
		}
	
		// validate the referenced Activity i.e. the inner Object - which is a Like-Activity but without start time
		if undo.Object == nil {
			result = append(result, "object should not be empty.")
		} else if activity, ok := undo.Object.(*ap.Activity); !ok {
			result = append(result, "object is not of type Activity")
		} else {
	
			result = append(result, validation.ValidateNotEmpty(string(activity.Type), "type")...)
			result = append(result, validation.ValidateOneOf(string(activity.Type), []any{"Like"}, "type")...)
	
			if activity.Actor == nil {
				result = append(result, "Object.Actor should not be nil.")
			} else {
				result = append(result, validation.ValidateNotEmpty(activity.Actor.GetID().String(), "actor")...)
			}
	
			if activity.Object == nil {
				result = append(result, "Object.Object should not be nil.")
			} else {
				result = append(result, validation.ValidateNotEmpty(activity.Object.GetID().String(), "object")...)
			}
		}
		return result
	}
	

	