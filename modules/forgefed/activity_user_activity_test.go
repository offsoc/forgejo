// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgefed

import (
	"testing"

	"forgejo.org/modules/validation"
	ap "github.com/go-ap/activitypub"
)

func Test_Validate(t *testing.T) {
	note := ForgeUserActivityNote{}
	note.Type = "Note"
	note.Content = ap.NaturalLanguageValues{
		{
			Ref:   ap.NilLangRef,
			Value: ap.Content("Any Content!"),
		},
	}
	note.URL = ap.IRI("example.org/user-id/57")

	if res, _ := validation.IsValid(note); !res {
		t.Errorf("sut expected to be valid: %v\n", note.Validate())
	}

	sut := ForgeUserActivity{}
	sut.Type = "Create"
	sut.Actor = ap.IRI("example.org/user-id/23")
	sut.CC = ap.ItemCollection{
		ap.IRI("example.org/registration/public#2nd"),
	}
	sut.To = ap.ItemCollection{
		ap.IRI("example.org/registration/public"),
	}

	sut.Note = note

	if res, _ := validation.IsValid(sut); !res {
		t.Errorf("sut expected to be valid: %v\n", sut.Validate())
	}
}
