// Copyright 2023, 2024, 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgefed

import (
	"testing"

	"forgejo.org/modules/setting"
)

func TestNewRepositoryId(t *testing.T) {
	setting.AppURL = "http://localhost:3000/"
	expected := RepositoryID{}
	expected.ID = "1"
	expected.Source = "forgejo"
	expected.HostSchema = "http"
	expected.Path = "api/activitypub/repository-id"
	expected.Host = "localhost"
	expected.HostPort = 3000
	expected.IsPortSupplemented = false
	expected.UnvalidatedInput = "http://localhost:3000/api/activitypub/repository-id/1"
	sut, _ := NewRepositoryID("http://localhost:3000/api/activitypub/repository-id/1", "forgejo")
	if sut != expected {
		t.Errorf("expected: %v\n but was: %v\n", expected, sut)
	}
}
