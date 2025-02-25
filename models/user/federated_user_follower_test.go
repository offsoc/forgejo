// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user

import (
	"testing"

	"code.gitea.io/gitea/modules/validation"
)

func Test_FederatedUserFollowerValidation(t *testing.T) {
	sut := FederatedUserFollower{
		LocalUserID:     12,
		FederatedUserID: 1,
	}
	if res, err := validation.IsValid(sut); !res {
		t.Errorf("sut should be valid but was %q", err)
	}

	sut = FederatedUserFollower{
		LocalUserID: 1,
	}
	if res, _ := validation.IsValid(sut); res {
		t.Errorf("sut should be invalid")
	}
}
