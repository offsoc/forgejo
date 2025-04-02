// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"testing"

	"forgejo.org/tests"
)

func TestEasyMDESwitch(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	session := loginUser(t, "user2")
	testEasyMDESwitch(t, session, "user2/glob/issues/1", false)
	testEasyMDESwitch(t, session, "user2/glob/issues/new", false)
	testEasyMDESwitch(t, session, "user2/glob/wiki?action=_new", true)
	testEasyMDESwitch(t, session, "user2/glob/releases/new", true)
	testEasyMDESwitch(t, session, "user2/glob/milestones/new", true)
	testEasyMDESwitch(t, session, "user2/repo1/milestones/1/edit", true)
}

func testEasyMDESwitch(t *testing.T, session *TestSession, url string, expected bool) {
	t.Helper()
	req := NewRequest(t, "GET", url)
	resp := session.MakeRequest(t, req, http.StatusOK)
	doc := NewHTMLParser(t, resp.Body)
	doc.AssertElement(t, ".combo-markdown-editor button.markdown-switch-easymde", expected)
}
