// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"bytes"
	"fmt"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"testing"

	"forgejo.org/models/db"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/avatar"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserAvatar(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2}) // owner of the repo3, is an org

	seed := user2.Email
	if len(seed) == 0 {
		seed = user2.Name
	}

	img, err := avatar.RandomImage([]byte(seed))
	if err != nil {
		require.NoError(t, err)
		return
	}

	session := loginUser(t, "user2")
	csrf := GetCSRF(t, session, "/user/settings")

	imgData := &bytes.Buffer{}

	body := &bytes.Buffer{}

	// Setup multi-part
	writer := multipart.NewWriter(body)
	writer.WriteField("source", "local")
	part, err := writer.CreateFormFile("avatar", "avatar-for-testuseravatar.png")
	if err != nil {
		require.NoError(t, err)
		return
	}

	if err := png.Encode(imgData, img); err != nil {
		require.NoError(t, err)
		return
	}

	if _, err := io.Copy(part, imgData); err != nil {
		require.NoError(t, err)
		return
	}

	if err := writer.Close(); err != nil {
		require.NoError(t, err)
		return
	}

	req := NewRequestWithBody(t, "POST", "/user/settings/avatar", body)
	req.Header.Add("X-Csrf-Token", csrf)
	req.Header.Add("Content-Type", writer.FormDataContentType())

	session.MakeRequest(t, req, http.StatusSeeOther)

	user2 = unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2}) // owner of the repo3, is an org

	req = NewRequest(t, "GET", user2.AvatarLinkWithSize(db.DefaultContext, 0))
	_ = session.MakeRequest(t, req, http.StatusOK)

	req = NewRequestf(t, "GET", "/%s.png", user2.Name)
	resp := MakeRequest(t, req, http.StatusSeeOther)
	assert.Equal(t, fmt.Sprintf("/avatars/%s", user2.Avatar), resp.Header().Get("location"))

	// Can't test if the response matches because the image is re-generated on upload but checking that this at least doesn't give a 404 should be enough.
}

func TestAvatarAnchorDestination(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// If the user is logged in, and looking at their own profile,
	// the avatar becomes a link towards the user settings page.
	// Test that the link does not show up when not viewing one's own profile,
	// and that, if the link does show up, there is a corresponding element
	// on the user settings page matching the fragment of the anchor.

	t.Run("viewing other's profile", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		profilePage := NewHTMLParser(t, MakeRequest(t, NewRequest(t, "GET", "/user2"), http.StatusOK).Body)
		profilePage.AssertElement(t, "#profile-avatar", true)
		// When viewing another user's profile, there shouldn't be a link to user settings
		profilePage.AssertElement(t, "#profile-avatar a", false)
	})

	t.Run("viewing own profile", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		session := loginUser(t, "user2")

		profilePage := NewHTMLParser(t, session.MakeRequest(t, NewRequest(t, "GET", "/user2"), http.StatusOK).Body)
		profilePage.AssertElement(t, "#profile-avatar a", true)
		href, has := profilePage.Find("#profile-avatar a").Attr("href")
		assert.True(t, has)

		settingsURL, err := url.Parse(href)
		require.NoError(t, err, "Change avatar link can't be parsed to URL")

		settingsPage := NewHTMLParser(t, session.MakeRequest(t, NewRequest(t, "GET", href), http.StatusOK).Body)
		settingsPage.AssertElement(t, fmt.Sprintf("#%s", settingsURL.Fragment), true)
	})
}
