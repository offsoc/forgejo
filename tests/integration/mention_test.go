// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHeadMentionCSS(t *testing.T) {
	userSession := loginUser(t, "user2")
	resp := userSession.MakeRequest(t, NewRequest(t, "GET", "/"), http.StatusOK)
	assert.Contains(t, resp.Body.String(), `.mention[href="/user2" i]`)

	guestSession := emptyTestSession(t)
	resp = guestSession.MakeRequest(t, NewRequest(t, "GET", "/"), http.StatusOK)
	assert.NotContains(t, resp.Body.String(), `.mention[href="`)
}
