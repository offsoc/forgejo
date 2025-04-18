// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"net/http"
	"testing"

	issues_model "forgejo.org/models/issues"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	forgejo_context "forgejo.org/services/context"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIssueAddTimeManually(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	session := loginUser(t, user2.Name)
	issue2 := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 2})
	require.NoError(t, issue2.LoadRepo(t.Context()))

	t.Run("No time", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		session.MakeRequest(t, NewRequestWithValues(t, "POST", issue2.Link()+"/times/add", map[string]string{
			"_csrf": GetCSRF(t, session, issue2.Link()),
		}), http.StatusSeeOther)

		flashCookie := session.GetCookie(forgejo_context.CookieNameFlash)
		assert.NotNil(t, flashCookie)
		assert.Contains(t, flashCookie.Value, "error%3DNo%2Btime%2Bwas%2Bentered.")
	})

	t.Run("Invalid hours", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		session.MakeRequest(t, NewRequestWithValues(t, "POST", issue2.Link()+"/times/add", map[string]string{
			"_csrf": GetCSRF(t, session, issue2.Link()),
			"hours": "-1",
		}), http.StatusSeeOther)

		flashCookie := session.GetCookie(forgejo_context.CookieNameFlash)
		assert.NotNil(t, flashCookie)
		assert.Contains(t, flashCookie.Value, "error%3DHours%2Bmust%2Bbe%2Ba%2Bnumber%2Bbetween%2B0%2Band%2B1%252C000.")
	})

	t.Run("Invalid minutes", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		session.MakeRequest(t, NewRequestWithValues(t, "POST", issue2.Link()+"/times/add", map[string]string{
			"_csrf":   GetCSRF(t, session, issue2.Link()),
			"minutes": "-1",
		}), http.StatusSeeOther)

		flashCookie := session.GetCookie(forgejo_context.CookieNameFlash)
		assert.NotNil(t, flashCookie)
		assert.Contains(t, flashCookie.Value, "error%3DMinutes%2Bmust%2Bbe%2Ba%2Bnumber%2Bbetween%2B0%2Band%2B1%252C000.")
	})

	t.Run("Normal", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		session.MakeRequest(t, NewRequestWithValues(t, "POST", issue2.Link()+"/times/add", map[string]string{
			"_csrf":   GetCSRF(t, session, issue2.Link()),
			"hours":   "3",
			"minutes": "14",
		}), http.StatusSeeOther)

		unittest.AssertExistsIf(t, true, &issues_model.TrackedTime{IssueID: issue2.ID, Time: 11640, UserID: user2.ID})
	})
}
