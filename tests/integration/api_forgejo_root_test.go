// Copyright The Forgejo Authors.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"testing"

	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
)

func TestAPIForgejoRoot(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	req := NewRequest(t, "GET", "/api/forgejo/v1")
	resp := MakeRequest(t, req, http.StatusNoContent)
	assert.Contains(t, resp.Header().Get("Link"), "/assets/forgejo/api.v1.yml")
}
