// Copyright 2024 The Forgejo Authors c/o Codeberg e.V.. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"image"
	"net/http"
	"testing"

	"forgejo.org/modules/setting"
	"forgejo.org/tests"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenGraphProperties(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	siteName := "Forgejo: Beyond coding. We Forge."

	cases := []struct {
		name     string
		url      string
		expected map[string]string
	}{
		{
			name: "website root",
			url:  "/",
			expected: map[string]string{
				"og:title":       siteName,
				"og:url":         setting.AppURL,
				"og:description": "Forgejo is a self-hosted lightweight software forge. Easy to install and low maintenance, it just does the job.",
				"og:type":        "website",
				"og:image":       "/assets/img/logo.png",
				"og:site_name":   siteName,
			},
		},
		{
			name: "profile page without description",
			url:  "/user30",
			expected: map[string]string{
				"og:title":     "User Thirty",
				"og:url":       setting.AppURL + "user30",
				"og:type":      "profile",
				"og:image":     setting.AppURL + "assets/img/avatar_default.png",
				"og:site_name": siteName,
			},
		},
		{
			name: "profile page with description",
			url:  "/the_34-user.with.all.allowedchars",
			expected: map[string]string{
				"og:title":       "the_1-user.with.all.allowedChars",
				"og:url":         setting.AppURL + "the_34-user.with.all.allowedChars",
				"og:description": "some [commonmark](https://commonmark.org/)!",
				"og:type":        "profile",
				"og:image":       setting.AppURL + "assets/img/avatar_default.png",
				"og:site_name":   siteName,
			},
		},
		{
			name: "issue",
			url:  "/user2/repo1/issues/1",
			expected: map[string]string{
				"og:title":        "issue1",
				"og:url":          setting.AppURL + "user2/repo1/issues/1",
				"og:description":  "content for the first issue",
				"og:type":         "object",
				"og:image":        setting.AppURL + "user2/repo1/issues/1/summary-card",
				"og:image:alt":    "Summary card of an issue titled \"issue1\" in repository user2/repo1",
				"og:image:width":  "1200",
				"og:image:height": "600",
				"og:site_name":    siteName,
			},
		},
		{
			name: "pull request",
			url:  "/user2/repo1/pulls/2",
			expected: map[string]string{
				"og:title":        "issue2",
				"og:url":          setting.AppURL + "user2/repo1/pulls/2",
				"og:description":  "content for the second issue",
				"og:type":         "object",
				"og:image":        setting.AppURL + "user2/repo1/pulls/2/summary-card",
				"og:image:alt":    "Summary card of an issue titled \"issue2\" in repository user2/repo1",
				"og:image:width":  "1200",
				"og:image:height": "600",
				"og:site_name":    siteName,
			},
		},
		{
			name: "file in repo",
			url:  "/user27/repo49/src/branch/master/test/test.txt",
			expected: map[string]string{
				"og:title":        "repo49/test/test.txt at master",
				"og:url":          setting.AppURL + "/user27/repo49/src/branch/master/test/test.txt",
				"og:type":         "object",
				"og:image":        setting.AppURL + "user27/repo49/-/summary-card",
				"og:image:alt":    "Summary card of repository user27/repo49",
				"og:image:width":  "1200",
				"og:image:height": "600",
				"og:site_name":    siteName,
			},
		},
		{
			name: "wiki page for repo without description",
			url:  "/user2/repo1/wiki/Page-With-Spaced-Name",
			expected: map[string]string{
				"og:title":        "Page With Spaced Name",
				"og:url":          setting.AppURL + "/user2/repo1/wiki/Page-With-Spaced-Name",
				"og:type":         "object",
				"og:image":        setting.AppURL + "user2/repo1/-/summary-card",
				"og:image:alt":    "Summary card of repository user2/repo1",
				"og:image:width":  "1200",
				"og:image:height": "600",
				"og:site_name":    siteName,
			},
		},
		{
			name: "index page for repo without description",
			url:  "/user2/repo1",
			expected: map[string]string{
				"og:title":        "repo1",
				"og:url":          setting.AppURL + "user2/repo1",
				"og:type":         "object",
				"og:image":        setting.AppURL + "user2/repo1/-/summary-card",
				"og:image:alt":    "Summary card of repository user2/repo1",
				"og:image:width":  "1200",
				"og:image:height": "600",
				"og:site_name":    siteName,
			},
		},
		{
			name: "index page for repo with description",
			url:  "/user27/repo49",
			expected: map[string]string{
				"og:title":        "repo49",
				"og:url":          setting.AppURL + "user27/repo49",
				"og:description":  "A wonderful repository with more than just a README.md",
				"og:type":         "object",
				"og:image":        setting.AppURL + "user27/repo49/-/summary-card",
				"og:image:alt":    "Summary card of repository user27/repo49",
				"og:image:width":  "1200",
				"og:image:height": "600",
				"og:site_name":    siteName,
			},
		},
		{
			name: "release",
			url:  "/user2/repo1/releases/tag/v1.1",
			expected: map[string]string{
				"og:title":        "testing-release - user2/repo1",
				"og:url":          setting.AppURL + "user2/repo1/releases/tag/v1.1",
				"og:type":         "object",
				"og:image":        setting.AppURL + "user2/repo1/releases/summary-card/v1.1",
				"og:image:alt":    "Summary card of an release titled \"testing-release\" in repository user2/repo1",
				"og:image:width":  "1200",
				"og:image:height": "600",
				"og:site_name":    siteName,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequest(t, "GET", tc.url)
			resp := MakeRequest(t, req, http.StatusOK)
			doc := NewHTMLParser(t, resp.Body)

			foundProps := make(map[string]string)
			doc.Find("head meta[property^=\"og:\"]").Each(func(_ int, selection *goquery.Selection) {
				prop, foundProp := selection.Attr("property")
				assert.True(t, foundProp)
				content, foundContent := selection.Attr("content")
				assert.True(t, foundContent, "opengraph meta tag without a content property")
				foundProps[prop] = content
			})

			assert.Equal(t, tc.expected, foundProps, "mismatching opengraph properties")
		})
	}
}

func TestOpenGraphSummaryCard(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	cases := []struct {
		name string
		url  string
	}{
		{
			name: "repo",
			url:  "/user2/repo1/-/summary-card",
		},
		{
			name: "issue",
			url:  "/user2/repo1/issues/1/summary-card",
		},
		{
			name: "pull request",
			url:  "/user2/repo1/pulls/2/summary-card",
		},
		{
			name: "release",
			url:  "/user2/repo1/releases/summary-card/v1.1",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequest(t, "GET", tc.url)
			resp := MakeRequest(t, req, http.StatusOK)

			assert.Equal(t, "image/png", resp.Header().Get("Content-Type"))
			img, imgType, err := image.Decode(resp.Body)
			require.NoError(t, err)
			assert.Equal(t, "png", imgType)
			assert.Equal(t, 1200, img.Bounds().Dx())
			assert.Equal(t, 600, img.Bounds().Dy())
		})
	}
}
