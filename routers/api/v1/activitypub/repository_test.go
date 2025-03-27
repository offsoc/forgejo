// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package activitypub

import (
	"testing"

	"forgejo.org/modules/validation"
)

func Test_UserEmailValidate(t *testing.T) {
	sut := "ab@cd.ef"
	if err := validation.ValidateEmail(sut); err != nil {
		t.Errorf("sut should be valid, %v, %v", sut, err)
	}

	sut = "83ce13c8-af0b-4112-8327-55a54e54e664@code.cartoon-aa.xyz"
	if err := validation.ValidateEmail(sut); err != nil {
		t.Errorf("sut should be valid, %v, %v", sut, err)
	}

	sut = "1"
	if err := validation.ValidateEmail(sut); err == nil {
		t.Errorf("sut should not be valid, %v", sut)
	}
}
