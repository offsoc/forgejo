// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"testing"

	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
)

func TestExploreRepos(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	req := NewRequest(t, "GET", "/explore/repos")
	MakeRequest(t, req, http.StatusOK)

	t.Run("Persistent parameters", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, "GET", "/explore/repos?topic=1&language=Go")
		resp := MakeRequest(t, req, http.StatusOK)
		htmlDoc := NewHTMLParser(t, resp.Body).Find("#repo-search-form")

		assert.Equal(t, "Go", htmlDoc.Find("input[name='language']").AttrOr("value", "not found"))
		assert.Equal(t, "true", htmlDoc.Find("input[name='topic']").AttrOr("value", "not found"))
	})
}
