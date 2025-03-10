// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"testing"

	"code.gitea.io/gitea/tests"

	"github.com/stretchr/testify/assert"
)

func TestViewMilestones(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	req := NewRequest(t, "GET", "/user2/repo1/milestones")
	resp := MakeRequest(t, req, http.StatusOK)

	htmlDoc := NewHTMLParser(t, resp.Body)
	search := htmlDoc.doc.Find(".list-header-search > .search > .input > input")
	placeholder, _ := search.Attr("placeholder")
	assert.Equal(t, "Search milestones…", placeholder)
}

func TestMilestonesCount(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	req := NewRequest(t, "GET", "/user2/repo1/milestones")
	resp := MakeRequest(t, req, http.StatusOK)

	htmlDoc := NewHTMLParser(t, resp.Body)

	openCount := htmlDoc.doc.Find("a[data-test-name='open-issue-count']").Text()
	assert.Contains(t, openCount, "2\u00a0Open")

	closedCount := htmlDoc.doc.Find("a[data-test-name='closed-issue-count']").Text()
	assert.Contains(t, closedCount, "1\u00a0Closed")

	assert.Empty(t, htmlDoc.doc.Find("a[data-test-name='all-issue-count']").Nodes)
}
