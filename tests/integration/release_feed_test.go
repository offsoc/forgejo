// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"net/http"
	"regexp"
	"testing"

	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
)

func TestReleaseFeed(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	normalize := func(body string) string {
		// Remove port.
		body = regexp.MustCompile(`localhost:\d+`).ReplaceAllString(body, "localhost")
		// date is timezone dependent.
		body = regexp.MustCompile(`<pubDate>.*</pubDate>`).ReplaceAllString(body, "<pubDate></pubDate>")
		body = regexp.MustCompile(`<updated>.*</updated>`).ReplaceAllString(body, "<updated></updated>")
		return body
	}
	t.Run("RSS feed", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		resp := MakeRequest(t, NewRequest(t, "GET", "/user2/repo1/releases.rss"), http.StatusOK)
		assert.Equal(t, `<?xml version="1.0" encoding="UTF-8"?><rss version="2.0" xmlns:content="http://purl.org/rss/1.0/modules/content/">
  <channel>
    <title>Releases for user2/repo1</title>
    <link>http://localhost/user2/repo1/release</link>
    <description></description>
    <pubDate></pubDate>
    <item>
      <title>pre-release</title>
      <link>http://localhost/user2/repo1/releases/tag/v1.0</link>
      <description></description>
      <content:encoded><![CDATA[<p dir="auto">some text for a pre release</p>
]]></content:encoded>
      <author>user2</author>
      <guid>5: http://localhost/user2/repo1/releases/tag/v1.0</guid>
      <pubDate></pubDate>
    </item>
    <item>
      <title>testing-release</title>
      <link>http://localhost/user2/repo1/releases/tag/v1.1</link>
      <description></description>
      <author>user2</author>
      <guid>1: http://localhost/user2/repo1/releases/tag/v1.1</guid>
      <pubDate></pubDate>
    </item>
  </channel>
</rss>`, normalize(resp.Body.String()))
	})

	t.Run("Atom feed", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		resp := MakeRequest(t, NewRequest(t, "GET", "/user2/repo1/releases.atom"), http.StatusOK)
		assert.Equal(t, `<?xml version="1.0" encoding="UTF-8"?><feed xmlns="http://www.w3.org/2005/Atom">
  <title>Releases for user2/repo1</title>
  <id>http://localhost/user2/repo1/release</id>
  <updated></updated>
  <link href="http://localhost/user2/repo1/release"></link>
  <entry>
    <title>pre-release</title>
    <updated></updated>
    <id>5: http://localhost/user2/repo1/releases/tag/v1.0</id>
    <content type="html">&lt;p dir=&#34;auto&#34;&gt;some text for a pre release&lt;/p&gt;&#xA;</content>
    <link href="http://localhost/user2/repo1/releases/tag/v1.0" rel="alternate"></link>
    <author>
      <name>user2</name>
      <email>user2@noreply.example.org</email>
    </author>
  </entry>
  <entry>
    <title>testing-release</title>
    <updated></updated>
    <id>1: http://localhost/user2/repo1/releases/tag/v1.1</id>
    <link href="http://localhost/user2/repo1/releases/tag/v1.1" rel="alternate"></link>
    <author>
      <name>user2</name>
      <email>user2@noreply.example.org</email>
    </author>
  </entry>
</feed>`, normalize(resp.Body.String()))
	})
}
