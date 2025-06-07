// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"forgejo.org/modules/json"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/test"
	"forgejo.org/routers"
	"forgejo.org/tests"

	"code.forgejo.org/go-chi/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getSessionID(t *testing.T, resp *httptest.ResponseRecorder) string {
	cookies := resp.Result().Cookies()
	found := false
	sessionID := ""
	for _, cookie := range cookies {
		if cookie.Name == setting.SessionConfig.CookieName {
			sessionID = cookie.Value
			found = true
		}
	}
	assert.True(t, found)
	assert.NotEmpty(t, sessionID)
	return sessionID
}

func sessionFile(tmpDir, sessionID string) string {
	return filepath.Join(tmpDir, sessionID[0:1], sessionID[1:2], sessionID)
}

func sessionFileExist(t *testing.T, tmpDir, sessionID string) bool {
	sessionFile := sessionFile(tmpDir, sessionID)
	_, err := os.Lstat(sessionFile)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
		require.NoError(t, err)
	}
	return true
}

func TestSessionFileCreation(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	defer test.MockProtect(&setting.SessionConfig.ProviderConfig)()
	defer test.MockProtect(&testWebRoutes)()

	var config session.Options
	err := json.Unmarshal([]byte(setting.SessionConfig.ProviderConfig), &config)
	require.NoError(t, err)

	config.Provider = "file"

	// Now create a temporaryDirectory
	tmpDir := t.TempDir()
	config.ProviderConfig = tmpDir

	newConfigBytes, err := json.Marshal(config)
	require.NoError(t, err)

	setting.SessionConfig.ProviderConfig = string(newConfigBytes)

	testWebRoutes = routers.NormalRoutes()

	t.Run("NoSessionOnViewIssue", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, "GET", "/user2/repo1/issues/1")
		resp := MakeRequest(t, req, http.StatusOK)
		sessionID := getSessionID(t, resp)

		// We're not logged in so there should be no session
		assert.False(t, sessionFileExist(t, tmpDir, sessionID))
	})
	t.Run("CreateSessionOnLogin", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, "GET", "/user/login")
		resp := MakeRequest(t, req, http.StatusOK)
		sessionID := getSessionID(t, resp)

		// We're not logged in so there should be no session
		assert.False(t, sessionFileExist(t, tmpDir, sessionID))

		doc := NewHTMLParser(t, resp.Body)
		req = NewRequestWithValues(t, "POST", "/user/login", map[string]string{
			"_csrf":     doc.GetCSRF(),
			"user_name": "user2",
			"password":  userPassword,
		})
		resp = MakeRequest(t, req, http.StatusSeeOther)
		sessionID = getSessionID(t, resp)

		assert.FileExists(t, sessionFile(tmpDir, sessionID))
	})
}
