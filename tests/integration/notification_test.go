// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"net/http"
	"testing"

	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/tests"
)

func TestNotification(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	session := loginUser(t, user2.Name)

	req := NewRequest(t, "GET", "/notifications")
	resp := session.MakeRequest(t, req, http.StatusOK)
	htmlDoc := NewHTMLParser(t, resp.Body)

	// Unread and pinned notification.
	htmlDoc.AssertElement(t, ".notifications-link[href='/user2/repo1/pulls/3']", true)
	htmlDoc.AssertElement(t, ".notifications-link[href='/user2/repo1/issues/4']", true)
	htmlDoc.AssertElement(t, ".notifications-link[href='/user2/repo2/issues/1']", true)

	// Read notification.
	htmlDoc.AssertElement(t, ".notifications-link[href='/user2/repo2/pulls/2']", false)
}
