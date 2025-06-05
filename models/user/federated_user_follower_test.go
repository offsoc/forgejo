// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user

import (
	"testing"

	"forgejo.org/modules/validation"
)

func Test_FederatedUserFollowerValidation(t *testing.T) {
	sut := FederatedUserFollower{
		FollowedUserID:  12,
		FollowingUserID: 1,
	}
	if res, err := validation.IsValid(sut); !res {
		t.Errorf("sut should be valid but was %q", err)
	}

	sut = FederatedUserFollower{
		FollowedUserID: 1,
	}
	if res, _ := validation.IsValid(sut); res {
		t.Errorf("sut should be invalid")
	}
}
