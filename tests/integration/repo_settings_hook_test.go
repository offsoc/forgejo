// Copyright 2022 The Forgejo Authors c/o Codeberg e.V.. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"strings"
	"testing"

	"forgejo.org/models/db"
	"forgejo.org/models/webhook"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepoSettingsHookHistory(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	_, err := db.GetEngine(t.Context()).ID(2).Cols("is_delivered").Update(&webhook.HookTask{IsDelivered: false})
	require.NoError(t, err)

	session := loginUser(t, "user2")

	// Request repository hook page with history
	req := NewRequest(t, "GET", "/user2/repo1/settings/hooks/1")
	resp := session.MakeRequest(t, req, http.StatusOK)

	doc := NewHTMLParser(t, resp.Body)

	t.Run("1/delivered", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		html, err := doc.doc.Find(".webhook div[data-tab='request-1']").Html()
		require.NoError(t, err)
		assert.Contains(t, html, "<strong>Request URL:</strong> /matrix-delivered\n")
		assert.Contains(t, html, "<strong>Request method:</strong> PUT")
		assert.Contains(t, html, "<strong>X-Head:</strong> 42")
		assert.Contains(t, html, `<code class="json">{}</code>`)

		val, ok := doc.doc.Find(".webhook div.item:has(div#info-1) svg").Attr("class")
		assert.True(t, ok)
		assert.Equal(t, "svg octicon-alert", val)
	})

	t.Run("2/undelivered", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		html, err := doc.doc.Find(".webhook div[data-tab='request-2']").Html()
		require.NoError(t, err)
		assert.Equal(t, "-", strings.TrimSpace(html))

		val, ok := doc.doc.Find(".webhook div.item:has(div#info-2) svg").Attr("class")
		assert.True(t, ok)
		assert.Equal(t, "svg octicon-stopwatch", val)
	})

	t.Run("3/success", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		html, err := doc.doc.Find(".webhook div[data-tab='request-3']").Html()
		require.NoError(t, err)
		assert.Contains(t, html, "<strong>Request URL:</strong> /matrix-success\n")
		assert.Contains(t, html, "<strong>Request method:</strong> PUT")
		assert.Contains(t, html, "<strong>X-Head:</strong> 42")
		assert.Contains(t, html, `<code class="json">{&#34;key&#34;:&#34;value&#34;}</code>`)

		val, ok := doc.doc.Find(".webhook div.item:has(div#info-3) svg").Attr("class")
		assert.True(t, ok)
		assert.Equal(t, "svg octicon-check", val)
	})
}
