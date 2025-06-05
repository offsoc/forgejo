// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package activities

import (
	"testing"

	"forgejo.org/modules/validation"
)

func Test_FederatedUserActivityValidation(t *testing.T) {
	sut := FederatedUserActivity{}
	sut.UserID = 13
	sut.ActorID = 33
	sut.ActorURI = "33"
	sut.NoteContent = "Any content!"
	sut.NoteURL = "https://example.org/note/17"
	sut.OriginalNote = "federatedUserActivityNote-17"

	if res, _ := validation.IsValid(sut); !res {
		t.Errorf("sut expected to be valid: %v\n", sut.Validate())
	}
}
