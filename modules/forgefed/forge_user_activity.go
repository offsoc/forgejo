// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgefed

import (
	"context"
	"fmt"
	"time"

	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/validation"

	ap "github.com/go-ap/activitypub"
)

type ForgeUserActivityNote struct {
	// swagger.ignore
	ap.Object
}

type ForgeUserActivity struct {
	ap.Activity
}

type ForgeUserFollowRequest struct {
	ActorID string `json:"actor_id"`
}

func newNote(doer *user_model.User, content, id string, published time.Time) (ForgeUserActivityNote, error) {
	note := ForgeUserActivityNote{}
	note.Type = ap.NoteType
	note.AttributedTo = ap.IRI(doer.APActorID())
	note.Content = ap.NaturalLanguageValues{
		{
			Ref:   ap.NilLangRef,
			Value: ap.Content(content),
		},
	}
	note.ID = ap.IRI(id)
	note.Published = published
	note.URL = ap.IRI(id)
	note.To = ap.ItemCollection{
		ap.IRI("https://www.w3.org/ns/activitystreams#Public"),
	}
	note.CC = ap.ItemCollection{
		ap.IRI(doer.APActorID() + "/followers"),
	}

	if valid, err := validation.IsValid(note); !valid {
		return ForgeUserActivityNote{}, err
	}

	return note, nil
}

func NewForgeUserActivity(ctx context.Context, doer *user_model.User, actionID int64, content string) (ForgeUserActivity, error) {
	id := fmt.Sprintf("%s/activities/%d", doer.APActorID(), actionID)
	published := time.Now()

	result := ForgeUserActivity{}
	result.ID = ap.IRI(id + "/activity")
	result.Type = ap.CreateType
	result.Actor = ap.IRI(doer.APActorID())
	result.Published = published
	result.To = ap.ItemCollection{
		ap.IRI("https://www.w3.org/ns/activitystreams#Public"),
	}
	result.CC = ap.ItemCollection{
		ap.IRI(doer.APActorID() + "/followers"),
	}
	note, err := newNote(doer, content, id, published)
	if err != nil {
		return ForgeUserActivity{}, err
	}
	result.Object = note

	return result, nil
}

func (note ForgeUserActivityNote) Validate() []string {
	var result []string
	result = append(result, validation.ValidateNotEmpty(string(note.Type), "type")...)
	result = append(result, validation.ValidateOneOf(string(note.Type), []any{"Note"}, "type")...)
	result = append(result, validation.ValidateNotEmpty(note.Content.String(), "content")...)
	if len(note.Content) == 0 {
		result = append(result, "Content was invalid.")
	}

	return result
}

func (userActivity ForgeUserActivity) Validate() []string {
	var result []string
	result = append(result, validation.ValidateNotEmpty(string(userActivity.Type), "type")...)
	result = append(result, validation.ValidateOneOf(string(userActivity.Type), []any{"Create"}, "type")...)
	result = append(result, validation.ValidateNotEmpty(string(userActivity.ID), "id")...)

	if len(userActivity.To) == 0 {
		result = append(result, "Missing To")
	}

	return result
}
