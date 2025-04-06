// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package validation

import (
	"testing"

	"forgejo.org/modules/timeutil"
)

type Sut struct {
	valid bool
}

func (sut Sut) Validate() []string {
	if sut.valid {
		return []string{}
	}
	return []string{"invalid"}
}

func Test_IsValid(t *testing.T) {
	sut := Sut{valid: true}
	if res, _ := IsValid(sut); !res {
		t.Errorf("sut expected to be valid: %v\n", sut.Validate())
	}
	sut = Sut{valid: false}
	res, err := IsValid(sut)
	if res {
		t.Errorf("sut expected to be invalid: %v\n", sut.Validate())
	}
	if err == nil || !IsErrNotValid(err) || err.Error() != "Validation Error: validation.Sut: invalid" {
		t.Errorf("validation error expected, but was %v", err)
	}
}

func Test_ValidateNotEmpty_ForString(t *testing.T) {
	sut := ""
	if len(ValidateNotEmpty(sut, "dummyField")) == 0 {
		t.Errorf("sut should be invalid")
	}
	sut = "not empty"
	if res := ValidateNotEmpty(sut, "dummyField"); len(res) > 0 {
		t.Errorf("sut should be valid but was %q", res)
	}
}

func Test_ValidateNotEmpty_ForTimestamp(t *testing.T) {
	sut := timeutil.TimeStamp(0)
	if res := ValidateNotEmpty(sut, "dummyField"); len(res) == 0 {
		t.Errorf("sut should be invalid")
	}
	sut = timeutil.TimeStampNow()
	if res := ValidateNotEmpty(sut, "dummyField"); len(res) > 0 {
		t.Errorf("sut should be valid but was %q", res)
	}
}

func Test_ValidateMaxLen(t *testing.T) {
	sut := "0123456789"
	if len(ValidateMaxLen(sut, 9, "dummyField")) == 0 {
		t.Errorf("sut should be invalid")
	}
	sut = "0123456789"
	if res := ValidateMaxLen(sut, 11, "dummyField"); len(res) > 0 {
		t.Errorf("sut should be valid but was %q", res)
	}
}
