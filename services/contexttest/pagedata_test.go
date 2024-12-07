// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package contexttest

import (
	"testing"

	"code.gitea.io/gitea/modules/translation"

	"github.com/stretchr/testify/assert"
)

func TestPluralStringsForClient(t *testing.T) {
	mockLocale := translation.MockLocale{}
	mockLocale.MockTranslations = map[string]string{
		"relativetime.mins":   "%d minute ago | %d minutes ago",
		"relativetime.hours":  "%d hour ago | %d hours ago",
		"relativetime.days":   "%d day ago | %d days ago",
		"relativetime.weeks":  "%d week ago | %d weeks ago",
		"relativetime.months": "%d month ago | %d months ago",
		"relativetime.years":  "%d year ago | %d years ago",
	}

	ctx, _ := MockContext(t, "/")
	ctx.Locale = mockLocale
	assert.True(t, ctx.Locale.HasKey("relativetime.mins"))
	assert.True(t, ctx.Locale.HasKey("relativetime.weeks"))
	assert.Equal(t, "%d minute ago | %d minutes ago", ctx.Locale.TrString("relativetime.mins"))
	assert.Equal(t, "%d week ago | %d weeks ago", ctx.Locale.TrString("relativetime.weeks"))

	assert.Empty(t, ctx.PageData)
	ctx.PageData["PLURALSTRINGS_LANG"] = map[string][]string{}
	assert.Empty(t, ctx.PageData["PLURALSTRINGS_LANG"])

	ctx.AddPluralStringsToPageData([]string{"relativetime.mins", "relativetime.hours"})
	assert.Len(t, ctx.PageData["PLURALSTRINGS_LANG"], 2)
	assert.Len(t, ctx.PageData["PLURALSTRINGS_LANG"].(map[string][]string)["relativetime.mins"], 2)
	assert.Len(t, ctx.PageData["PLURALSTRINGS_LANG"].(map[string][]string)["relativetime.hours"], 2)
	assert.Equal(t, "%d minute ago", ctx.PageData["PLURALSTRINGS_LANG"].(map[string][]string)["relativetime.mins"][0])
	assert.Equal(t, "%d minutes ago", ctx.PageData["PLURALSTRINGS_LANG"].(map[string][]string)["relativetime.mins"][1])
	assert.Equal(t, "%d hour ago", ctx.PageData["PLURALSTRINGS_LANG"].(map[string][]string)["relativetime.hours"][0])
	assert.Equal(t, "%d hours ago", ctx.PageData["PLURALSTRINGS_LANG"].(map[string][]string)["relativetime.hours"][1])

	ctx.AddPluralStringsToPageData([]string{"relativetime.years", "relativetime.days"})
	assert.Len(t, ctx.PageData["PLURALSTRINGS_LANG"], 4)
	assert.Len(t, ctx.PageData["PLURALSTRINGS_LANG"].(map[string][]string)["relativetime.mins"], 2)
	assert.Len(t, ctx.PageData["PLURALSTRINGS_LANG"].(map[string][]string)["relativetime.days"], 2)
	assert.Len(t, ctx.PageData["PLURALSTRINGS_LANG"].(map[string][]string)["relativetime.years"], 2)
	assert.Equal(t, "%d minute ago", ctx.PageData["PLURALSTRINGS_LANG"].(map[string][]string)["relativetime.mins"][0])
	assert.Equal(t, "%d minutes ago", ctx.PageData["PLURALSTRINGS_LANG"].(map[string][]string)["relativetime.mins"][1])
	assert.Equal(t, "%d day ago", ctx.PageData["PLURALSTRINGS_LANG"].(map[string][]string)["relativetime.days"][0])
	assert.Equal(t, "%d days ago", ctx.PageData["PLURALSTRINGS_LANG"].(map[string][]string)["relativetime.days"][1])
	assert.Equal(t, "%d year ago", ctx.PageData["PLURALSTRINGS_LANG"].(map[string][]string)["relativetime.years"][0])
	assert.Equal(t, "%d years ago", ctx.PageData["PLURALSTRINGS_LANG"].(map[string][]string)["relativetime.years"][1])
}
