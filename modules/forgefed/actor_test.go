// Copyright 2023, 2024, 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgefed

import (
	"testing"
)

func TestActorIdValidation(t *testing.T) {
	sut := ActorID{}
	sut.Source = "forgejo"
	sut.HostSchema = "https"
	sut.Path = "api/v1/activitypub/user-id"
	sut.Host = "an.other.host"
	sut.HostPort = 443
	sut.IsPortSupplemented = true
	sut.UnvalidatedInput = "https://an.other.host/api/v1/activitypub/user-id/"
	if sut.Validate()[0] != "userId should not be empty" {
		t.Errorf("validation error expected but was: %v\n", sut.Validate())
	}

	sut = ActorID{}
	sut.ID = "1"
	sut.Source = "forgejo"
	sut.HostSchema = "https"
	sut.Path = "api/v1/activitypub/user-id"
	sut.Host = "an.other.host"
	sut.HostPort = 443
	sut.IsPortSupplemented = true
	sut.UnvalidatedInput = "https://an.other.host/api/v1/activitypub/user-id/1?illegal=action"
	if sut.Validate()[0] != "not all input was parsed, \nUnvalidated Input:\"https://an.other.host/api/v1/activitypub/user-id/1?illegal=action\" \nParsed URI: \"https://an.other.host/api/v1/activitypub/user-id/1\"" {
		t.Errorf("validation error expected but was: %v\n", sut.Validate()[0])
	}
}
