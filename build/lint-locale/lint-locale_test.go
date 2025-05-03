// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT
package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLocalizationPolicy(t *testing.T) {
	initBlueMondayPolicy()
	initRemoveTags()

	t.Run("Remove tags", func(t *testing.T) {
		assert.Empty(t, checkLocaleContent([]byte(`hidden_comment_types_description = Comment types checked here will not be shown inside issue pages. Checking "Label" for example removes all "<user> added/removed <label>" comments.`)))

		assert.Equal(t, []string{"key: \x1b[31m<not-an-allowed-key>\x1b[0m REPLACED-TAG"}, checkLocaleContent([]byte(`key = "<not-an-allowed-key> <label>"`)))
		assert.Equal(t, []string{"key: \x1b[31m<user@example.com>\x1b[0m REPLACED-TAG"}, checkLocaleContent([]byte(`key = "<user@example.com> <email@example.com>"`)))
		assert.Equal(t, []string{"key: \x1b[31m<tag>\x1b[0m REPLACED-TAG \x1b[31m</tag>\x1b[0m"}, checkLocaleContent([]byte(`key = "<tag> <email@example.com> </tag>"`)))
	})

	t.Run("Specific exception", func(t *testing.T) {
		assert.Empty(t, checkLocaleContent([]byte(`workflow.dispatch.trigger_found = This workflow has a <c>workflow_dispatch</c> event trigger.`)))
		assert.Empty(t, checkLocaleContent([]byte(`pulls.title_desc_one = wants to merge %[1]d commit from <code>%[2]s</code> into <code id="%[4]s">%[3]s</code>`)))
		assert.Empty(t, checkLocaleContent([]byte(`editor.commit_directly_to_this_branch = Commit directly to the <strong class="%[2]s">%[1]s</strong> branch.`)))

		assert.Equal(t, []string{"workflow.dispatch.trigger_found: This workflow has a \x1b[31m<d>\x1b[0mworkflow_dispatch\x1b[31m</d>\x1b[0m event trigger."}, checkLocaleContent([]byte(`workflow.dispatch.trigger_found = This workflow has a <d>workflow_dispatch</d> event trigger.`)))
		assert.Equal(t, []string{"key: <code\x1b[31m id=\"branch_targe\"\x1b[0m>%[3]s</code>"}, checkLocaleContent([]byte(`key = <code id="branch_targe">%[3]s</code>`)))
		assert.Equal(t, []string{"key: <a\x1b[31m class=\"ui sh\"\x1b[0m href=\"https://TO-BE-REPLACED.COM\">"}, checkLocaleContent([]byte(`key = <a class="ui sh" href="%[3]s">`)))
		assert.Equal(t, []string{"key: <a\x1b[31m class=\"js-click-me\"\x1b[0m href=\"https://TO-BE-REPLACED.COM\">"}, checkLocaleContent([]byte(`key = <a class="js-click-me" href="%[3]s">`)))
		assert.Equal(t, []string{"key: <strong\x1b[31m class=\"branch-target\"\x1b[0m>%[1]s</strong>"}, checkLocaleContent([]byte(`key = <strong class="branch-target">%[1]s</strong>`)))
	})

	t.Run("General safe tags", func(t *testing.T) {
		assert.Empty(t, checkLocaleContent([]byte("error404 = The page you are trying to reach either <strong>does not exist</strong> or <strong>you are not authorized</strong> to view it.")))
		assert.Empty(t, checkLocaleContent([]byte("teams.specific_repositories_helper = Members will only have access to repositories explicitly added to the team. Selecting this <strong>will not</strong> automatically remove repositories already added with <i>All repositories</i>.")))
		assert.Empty(t, checkLocaleContent([]byte("sqlite_helper = File path for the SQLite3 database.<br>Enter an absolute path if you run Forgejo as a service.")))
		assert.Empty(t, checkLocaleContent([]byte("hi_user_x = Hi <b>%s</b>,")))

		assert.Equal(t, []string{"error404: The page you are trying to reach either <strong\x1b[31m title='aaa'\x1b[0m>does not exist</strong> or <strong>you are not authorized</strong> to view it."}, checkLocaleContent([]byte("error404 = The page you are trying to reach either <strong title='aaa'>does not exist</strong> or <strong>you are not authorized</strong> to view it.")))
	})

	t.Run("<a>", func(t *testing.T) {
		assert.Empty(t, checkLocaleContent([]byte(`admin.new_user.text = Please <a href="%s">click here</a> to manage this user from the admin panel.`)))
		assert.Empty(t, checkLocaleContent([]byte(`access_token_desc = Selected token permissions limit authorization only to the corresponding <a href="%[1]s" target="_blank">API</a> routes. Read the <a href="%[2]s" target="_blank">documentation</a> for more information.`)))
		assert.Empty(t, checkLocaleContent([]byte(`webauthn_desc = Security keys are hardware devices containing cryptographic keys. They can be used for two-factor authentication. Security keys must support the <a rel="noreferrer" target="_blank" href="%s">WebAuthn Authenticator</a> standard.`)))
		assert.Empty(t, checkLocaleContent([]byte("issues.closed_at = `closed this issue <a id=\"%[1]s\" href=\"#%[1]s\">%[2]s</a>`")))

		assert.Equal(t, []string{"key: \x1b[31m<a href=\"https://example.com\">\x1b[0m"}, checkLocaleContent([]byte(`key = <a href="https://example.com">`)))
		assert.Equal(t, []string{"key: \x1b[31m<a href=\"javascript:alert('1')\">\x1b[0m"}, checkLocaleContent([]byte(`key = <a href="javascript:alert('1')">`)))
		assert.Equal(t, []string{"key: <a href=\"https://TO-BE-REPLACED.COM\"\x1b[31m download\x1b[0m>"}, checkLocaleContent([]byte(`key = <a href="%s" download>`)))
		assert.Equal(t, []string{"key: <a href=\"https://TO-BE-REPLACED.COM\"\x1b[31m target=\"_self\"\x1b[0m>"}, checkLocaleContent([]byte(`key = <a href="%s" target="_self">`)))
		assert.Equal(t, []string{"key: \x1b[31m<a href=\"https://example.com/%s\">\x1b[0m"}, checkLocaleContent([]byte(`key = <a href="https://example.com/%s">`)))
		assert.Equal(t, []string{"key: \x1b[31m<a href=\"https://example.com/?q=%s\">\x1b[0m"}, checkLocaleContent([]byte(`key = <a href="https://example.com/?q=%s">`)))
		assert.Equal(t, []string{"key: \x1b[31m<a href=\"%s/open-redirect\">\x1b[0m"}, checkLocaleContent([]byte(`key = <a href="%s/open-redirect">`)))
		assert.Equal(t, []string{"key: \x1b[31m<a href=\"%s?q=open-redirect\">\x1b[0m"}, checkLocaleContent([]byte(`key = <a href="%s?q=open-redirect">`)))
	})

	t.Run("Escaped HTML characters", func(t *testing.T) {
		assert.Empty(t, checkLocaleContent([]byte("activity.git_stats_push_to_branch = `إلى %s و\"`")))

		assert.Equal(t, []string{"key: و\x1b[31m&nbsp;\x1b[0m\x1b[32m\u00a0\x1b[0m"}, checkLocaleContent([]byte(`key = و&nbsp;`)))
	})
}

func TestNextLocalizationPolicy(t *testing.T) {
	initBlueMondayPolicy()
	initRemoveTags()

	t.Run("Nested locales", func(t *testing.T) {
		assert.Empty(t, checkLocaleNextContent([]byte(`{
			"settings": {
				"hidden_comment_types_description": "Comment types checked here will not be shown inside issue pages. Checking \"Label\" for example removes all \"<user> added/removed <label>\" comments."
			}
		}`)))

		assert.Equal(t, []string{"settings.hidden_comment_types_description: \"\x1b[31m<not-an-allowed-key>\x1b[0m REPLACED-TAG\""}, checkLocaleNextContent([]byte(`{
			"settings": {
				"hidden_comment_types_description": "\"<not-an-allowed-key> <label>\""
			}
		}`)))
	})

	t.Run("Flat locales", func(t *testing.T) {
		assert.Empty(t, checkLocaleNextContent([]byte(`{
			"settings.hidden_comment_types_description": "Comment types checked here will not be shown inside issue pages. Checking \"Label\" for example removes all \"<user> added/removed <label>\" comments."
		}`)))

		assert.Equal(t, []string{"settings.hidden_comment_types_description: \"\x1b[31m<not-an-allowed-key>\x1b[0m REPLACED-TAG\""}, checkLocaleNextContent([]byte(`{
			"settings.hidden_comment_types_description": "\"<not-an-allowed-key> <label>\""
		}`)))
	})

	t.Run("Plural form", func(t *testing.T) {
		assert.Equal(t, []string{"repo.pulls.title_desc: key = \x1b[31m<a href=\"https://example.com\">\x1b[0m"}, checkLocaleNextContent([]byte(`{"repo.pulls.title_desc": {
                       "few": "key = <a href=\"%s\">",
                       "other": "key = <a href=\"https://example.com\">"
    }}`)))

		assert.Equal(t, []string{"repo.pulls.title_desc.few: key = \x1b[31m<a href=\"https://example.com\">\x1b[0m"}, checkLocaleNextContent([]byte(`{"repo.pulls.title_desc": {
                       "few": "key = <a href=\"https://example.com\">",
                       "other": "key = <a href=\"%s\">"
    }}`)))
	})
}
