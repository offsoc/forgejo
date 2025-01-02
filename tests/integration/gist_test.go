// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"net/http"
	"testing"

	"code.gitea.io/gitea/tests"
	"github.com/stretchr/testify/assert"
)

func TestViewGist(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	session := loginUser(t, "user2")

	t.Run("PublicGist", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		resp := MakeRequest(t, NewRequest(t, "GET", "/gists/df852aec"), http.StatusOK)

		body := resp.Body.String()
		assert.Contains(t, body, "<b>test.txt</b>")
	})

	t.Run("HiddenGist", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		resp := MakeRequest(t, NewRequest(t, "GET", "/gists/dec037f3"), http.StatusOK)

		body := resp.Body.String()
		assert.Contains(t, body, "<meta name=\"robots\" content=\"noindex\">")
		assert.Contains(t, body, "<b>a.txt</b>")
		assert.Contains(t, body, "<b>b.txt</b>")
	})

	t.Run("PrivateGist", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		MakeRequest(t, NewRequest(t, "GET", "/gists/261eccb7"), http.StatusNotFound)
		resp := session.MakeRequest(t, NewRequest(t, "GET", "/gists/261eccb7"), http.StatusOK)

		body := resp.Body.String()
		assert.Contains(t, body, "<b>test.py</b>")
	})
}
