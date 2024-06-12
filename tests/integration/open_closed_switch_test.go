// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOpenClosedSwitch(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, giteaURL *url.URL) {
		// Everything can be done in one test env and only one user is needed
		session := loginUser(t, "user5")

		session.MakeRequest(t, NewRequestWithValues(t, "POST", "/repo/create", map[string]string{
			"_csrf":     GetCSRF(t, session, "/repo/create"),
			"repo_name": "testing-issues",
		}), 303)

		//testOCSwitchGlobalIssues(t, session, "0 open ii", "0 closed ii")

		//testNewIssue(t, session, "user5", "testing-issues", "Switch test - issue 1", "")
		//testOCSwitchGlobalIssues(t, session, "1 open i", "0 closed ii")

		//testNewIssue(t, session, "user5", "testing-issues", "Switch test - issue 2", "")
		//testOCSwitchGlobalIssues(t, session, "2 open ii", "0 closed ii")

		//testIssueClose(t, session, "user5", "testing-issues")

		//testOCSwitchGlobalPulls(t, session)
		//testOCSwitchGlobalMilestones(t, session)
	})
}

func testOCSwitchGlobalIssues(t *testing.T, session *TestSession, expectedOpen, expectedClose string) {
	t.Helper()

	resp := session.MakeRequest(t, NewRequest(t, "GET", "/issues"), http.StatusOK)
	assert.EqualValues(t, expectedOpen, strings.TrimSpace(NewHTMLParser(t, resp.Body).Find(".list-header-toggle a[href*='state=open']").Text()))
	assert.EqualValues(t, expectedClose, strings.TrimSpace(NewHTMLParser(t, resp.Body).Find(".list-header-toggle a[href*='state=closed']").Text()))
}

func testOCSwitchGlobalPulls(t *testing.T, session *TestSession) {
	t.Helper()
}

func testOCSwitchGlobalMilestones(t *testing.T, session *TestSession) {
	t.Helper()
}
