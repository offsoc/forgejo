// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"testing"

	"code.gitea.io/gitea/models/db"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFeed(t *testing.T) {
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
		})
	})

	t.Run("Repo", func(t *testing.T) {
		t.Run("Normal", func(t *testing.T) {
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
}
