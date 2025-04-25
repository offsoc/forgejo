// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"encoding/xml"
	"net/http"
	"testing"

	"forgejo.org/models/db"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// RSS is a struct to unmarshal RSS feeds test only
type RSS struct {
	Channel struct {
		Title       string `xml:"title"`
		Link        string `xml:"link"`
		Description string `xml:"description"`
		PubDate     string `xml:"pubDate"`
		Items       []struct {
			Title       string `xml:"title"`
			Link        string `xml:"link"`
			Description string `xml:"description"`
			PubDate     string `xml:"pubDate"`
		} `xml:"item"`
	} `xml:"channel"`
}

func TestFeed(t *testing.T) {
	defer unittest.OverrideFixtures("tests/integration/fixtures/TestFeed")()
	defer tests.PrepareTestEnv(t)()

	t.Run("User", func(t *testing.T) {
		t.Run("Atom", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequest(t, "GET", "/user2.atom")
			resp := MakeRequest(t, req, http.StatusOK)

			data := resp.Body.String()
			assert.Contains(t, data, `<feed xmlns="http://www.w3.org/2005/Atom"`)
		})

		t.Run("RSS", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequest(t, "GET", "/user2.rss")
			resp := MakeRequest(t, req, http.StatusOK)

			data := resp.Body.String()
			assert.Contains(t, data, `<rss version="2.0"`)

			var rss RSS
			err := xml.Unmarshal(resp.Body.Bytes(), &rss)
			require.NoError(t, err)
			assert.Contains(t, rss.Channel.Link, "/user2")
			assert.NotEmpty(t, rss.Channel.PubDate)
		})
	})

	t.Run("Repo", func(t *testing.T) {
		t.Run("Normal", func(t *testing.T) {
			t.Run("Atom", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				req := NewRequest(t, "GET", "/user2/repo1.atom")
				resp := MakeRequest(t, req, http.StatusOK)

				data := resp.Body.String()
				assert.Contains(t, data, `<feed xmlns="http://www.w3.org/2005/Atom"`)
				assert.Contains(t, data, "This is a very long text, so lets scream together: aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
			})
			t.Run("RSS", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				req := NewRequest(t, "GET", "/user2/repo1.rss")
				resp := MakeRequest(t, req, http.StatusOK)

				data := resp.Body.String()
				assert.Contains(t, data, `<rss version="2.0"`)
				assert.Contains(t, data, "This is a very long text, so lets scream together: aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
			})
		})
		t.Run("Branch", func(t *testing.T) {
			t.Run("Atom", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				req := NewRequest(t, "GET", "/user2/repo1/atom/branch/master")
				resp := MakeRequest(t, req, http.StatusOK)

				data := resp.Body.String()
				assert.Contains(t, data, `<feed xmlns="http://www.w3.org/2005/Atom"`)
			})
			t.Run("RSS", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				req := NewRequest(t, "GET", "/user2/repo1/rss/branch/master")
				resp := MakeRequest(t, req, http.StatusOK)

				data := resp.Body.String()
				assert.Contains(t, data, `<rss version="2.0"`)
			})
		})
		t.Run("Empty", func(t *testing.T) {
			err := user_model.UpdateUserCols(db.DefaultContext, &user_model.User{ID: 30, ProhibitLogin: false}, "prohibit_login")
			require.NoError(t, err)

			session := loginUser(t, "user30")
			t.Run("Atom", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				req := NewRequest(t, "GET", "/user30/empty/atom/branch/master")
				session.MakeRequest(t, req, http.StatusNotFound)

				req = NewRequest(t, "GET", "/user30/empty.atom/src/branch/master")
				session.MakeRequest(t, req, http.StatusNotFound)
			})
			t.Run("RSS", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				req := NewRequest(t, "GET", "/user30/empty/rss/branch/master")
				session.MakeRequest(t, req, http.StatusNotFound)

				req = NewRequest(t, "GET", "/user30/empty.rss/src/branch/master")
				session.MakeRequest(t, req, http.StatusNotFound)
			})
		})
	})

	t.Run("View permission", func(t *testing.T) {
		t.Run("Anonymous", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			req := NewRequest(t, "GET", "/org3/repo3/rss/branch/master")
			MakeRequest(t, req, http.StatusNotFound)
		})
		t.Run("No code permission", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			session := loginUser(t, "user8")
			req := NewRequest(t, "GET", "/org3/repo3/rss/branch/master")
			session.MakeRequest(t, req, http.StatusNotFound)
		})
		t.Run("With code permission", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			session := loginUser(t, "user9")
			req := NewRequest(t, "GET", "/org3/repo3/rss/branch/master")
			session.MakeRequest(t, req, http.StatusOK)
		})
	})
}
