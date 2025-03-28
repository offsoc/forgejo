// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"net/url"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIssueSubscribe(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, giteaURL *url.URL) {
		session := emptyTestSession(t)
		testIssueSubscribe(t, *session, true)
	})
}

func testIssueSubscribe(t *testing.T, session TestSession, unavailable bool) {
	t.Helper()

	testIssue := "/user2/repo1/issues/1"
	testPull := "/user2/repo1/pulls/2"
	selector := ".issue-content-right .watching form"

	resp := session.MakeRequest(t, NewRequest(t, "GET", path.Join(testIssue)), http.StatusOK)
	area := NewHTMLParser(t, resp.Body).Find(selector)
	tooltip, exists := area.Attr("data-tooltip-content")
	assert.Equal(t, unavailable, exists)
	if unavailable {
		assert.Equal(t, "Sign in to subscribe to this issue.", tooltip)
	}

	resp = session.MakeRequest(t, NewRequest(t, "GET", path.Join(testPull)), http.StatusOK)
	area = NewHTMLParser(t, resp.Body).Find(selector)
	tooltip, exists = area.Attr("data-tooltip-content")
	assert.Equal(t, unavailable, exists)
	if unavailable {
		assert.Equal(t, "Sign in to subscribe to this pull request.", tooltip)
	}
}
