// Copyright 2024 The Forgejo Authors c/o Codeberg e.V.. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	auth_model "forgejo.org/models/auth"
	"forgejo.org/models/db"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/setting"
	api "forgejo.org/modules/structs"
	"forgejo.org/modules/test"
	"forgejo.org/routers"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
)

func TestRepositoryFlagsUIDisabled(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	defer test.MockVariableValue(&setting.Repository.EnableFlags, false)()
	defer test.MockVariableValue(&testWebRoutes, routers.NormalRoutes())()

	admin := unittest.AssertExistsAndLoadBean(t, &user_model.User{IsAdmin: true})
	session := loginUser(t, admin.Name)

	// With the repo flags feature disabled, the /flags route is 404
	req := NewRequest(t, "GET", "/user2/repo1/flags")
	session.MakeRequest(t, req, http.StatusNotFound)

	// With the repo flags feature disabled, the "Modify flags" tab does not
	// appear for instance admins
	req = NewRequest(t, "GET", "/user2/repo1")
	resp := session.MakeRequest(t, req, http.StatusOK)
	doc := NewHTMLParser(t, resp.Body)
	flagsLinkCount := doc.Find(fmt.Sprintf(`a[href="%s/flags"]`, "/user2/repo1")).Length()
	assert.Equal(t, 0, flagsLinkCount)
}

func TestRepositoryFlagsAPI(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	defer test.MockVariableValue(&setting.Repository.EnableFlags, true)()
	defer test.MockVariableValue(&testWebRoutes, routers.NormalRoutes())()

	// *************
	// ** Helpers **
	// *************

	adminUser := unittest.AssertExistsAndLoadBean(t, &user_model.User{IsAdmin: true}).Name
	normalUserBean := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	assert.False(t, normalUserBean.IsAdmin)
	normalUser := normalUserBean.Name

	assertAccess := func(t *testing.T, user, method, uri string, expectedStatus int) {
		session := loginUser(t, user)
		token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeReadAdmin)

		req := NewRequestf(t, method, "/api/v1/repos/user2/repo1/flags%s", uri).AddTokenAuth(token)
		MakeRequest(t, req, expectedStatus)
	}

	// ***********
	// ** Tests **
	// ***********

	t.Run("API access", func(t *testing.T) {
		t.Run("as admin", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			assertAccess(t, adminUser, "GET", "", http.StatusOK)
		})

		t.Run("as normal user", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			assertAccess(t, normalUser, "GET", "", http.StatusForbidden)
		})
	})

	t.Run("token scopes", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		// Trying to access the API with a token that lacks permissions, will
		// fail, even if the token owner is an instance admin.
		session := loginUser(t, adminUser)
		token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository)

		req := NewRequest(t, "GET", "/api/v1/repos/user2/repo1/flags").AddTokenAuth(token)
		MakeRequest(t, req, http.StatusForbidden)
	})

	t.Run("setting.Repository.EnableFlags is respected", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		defer test.MockVariableValue(&setting.Repository.EnableFlags, false)()
		defer test.MockVariableValue(&testWebRoutes, routers.NormalRoutes())()

		t.Run("as admin", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			assertAccess(t, adminUser, "GET", "", http.StatusNotFound)
		})

		t.Run("as normal user", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			assertAccess(t, normalUser, "GET", "", http.StatusNotFound)
		})
	})

	t.Run("API functionality", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 4})
		defer func() {
			repo.ReplaceAllFlags(db.DefaultContext, []string{})
		}()

		baseURLFmtStr := "/api/v1/repos/user5/repo4/flags%s"

		session := loginUser(t, adminUser)
		token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteAdmin)

		// Listing flags
		req := NewRequestf(t, "GET", baseURLFmtStr, "").AddTokenAuth(token)
		resp := MakeRequest(t, req, http.StatusOK)
		var flags []string
		DecodeJSON(t, resp, &flags)
		assert.Empty(t, flags)

		// Replacing all tags works, twice in a row
		for i := 0; i < 2; i++ {
			req = NewRequestWithJSON(t, "PUT", fmt.Sprintf(baseURLFmtStr, ""), &api.ReplaceFlagsOption{
				Flags: []string{"flag-1", "flag-2", "flag-3"},
			}).AddTokenAuth(token)
			MakeRequest(t, req, http.StatusNoContent)
		}

		// The list now includes all three flags
		req = NewRequestf(t, "GET", baseURLFmtStr, "").AddTokenAuth(token)
		resp = MakeRequest(t, req, http.StatusOK)
		DecodeJSON(t, resp, &flags)
		assert.Len(t, flags, 3)
		for _, flag := range []string{"flag-1", "flag-2", "flag-3"} {
			assert.True(t, slices.Contains(flags, flag))
		}

		// Check a flag that is on the repo
		req = NewRequestf(t, "GET", baseURLFmtStr, "/flag-1").AddTokenAuth(token)
		MakeRequest(t, req, http.StatusNoContent)

		// Check a flag that isn't on the repo
		req = NewRequestf(t, "GET", baseURLFmtStr, "/no-such-flag").AddTokenAuth(token)
		MakeRequest(t, req, http.StatusNotFound)

		// We can add the same flag twice
		for i := 0; i < 2; i++ {
			req = NewRequestf(t, "PUT", baseURLFmtStr, "/brand-new-flag").AddTokenAuth(token)
			MakeRequest(t, req, http.StatusNoContent)
		}

		// The new flag is there
		req = NewRequestf(t, "GET", baseURLFmtStr, "/brand-new-flag").AddTokenAuth(token)
		MakeRequest(t, req, http.StatusNoContent)

		// We can delete a flag, twice
		for i := 0; i < 2; i++ {
			req = NewRequestf(t, "DELETE", baseURLFmtStr, "/flag-3").AddTokenAuth(token)
			MakeRequest(t, req, http.StatusNoContent)
		}

		// We can delete a flag that wasn't there
		req = NewRequestf(t, "DELETE", baseURLFmtStr, "/no-such-flag").AddTokenAuth(token)
		MakeRequest(t, req, http.StatusNoContent)

		// We can delete all of the flags in one go, too
		req = NewRequestf(t, "DELETE", baseURLFmtStr, "").AddTokenAuth(token)
		MakeRequest(t, req, http.StatusNoContent)

		// ..once all flags are deleted, none are listed, either
		req = NewRequestf(t, "GET", baseURLFmtStr, "").AddTokenAuth(token)
		resp = MakeRequest(t, req, http.StatusOK)
		DecodeJSON(t, resp, &flags)
		assert.Empty(t, flags)
	})
}

func TestRepositoryFlagsUI(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	defer test.MockVariableValue(&setting.Repository.EnableFlags, true)()
	defer test.MockVariableValue(&testWebRoutes, routers.NormalRoutes())()

	// *******************
	//  ** Preparations **
	// *******************
	flaggedRepo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	unflaggedRepo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 4})

	// **************
	//  ** Helpers **
	// **************

	adminUser := unittest.AssertExistsAndLoadBean(t, &user_model.User{IsAdmin: true}).Name
	flaggedOwner := "user2"
	flaggedRepoURLStr := "/user2/repo1"
	unflaggedOwner := "user5"
	unflaggedRepoURLStr := "/user5/repo4"
	otherUser := "user4"

	ensureFlags := func(repo *repo_model.Repository, flags []string) func() {
		repo.ReplaceAllFlags(db.DefaultContext, flags)

		return func() {
			repo.ReplaceAllFlags(db.DefaultContext, flags)
		}
	}

	// Tests:
	// - Presence of the link
	// - Number of flags listed in the admin-only message box
	// - Whether there's a link to /user/repo/flags
	// - Whether /user/repo/flags is OK or Forbidden
	assertFlagAccessAndCount := func(t *testing.T, user, repoURL string, hasAccess bool, expectedFlagCount int) {
		t.Helper()

		var expectedLinkCount int
		var expectedStatus int
		if hasAccess {
			expectedLinkCount = 1
			expectedStatus = http.StatusOK
		} else {
			expectedLinkCount = 0
			if user != "" {
				expectedStatus = http.StatusForbidden
			} else {
				expectedStatus = http.StatusSeeOther
			}
		}

		var resp *httptest.ResponseRecorder
		var session *TestSession
		req := NewRequest(t, "GET", repoURL)
		if user != "" {
			session = loginUser(t, user)
			resp = session.MakeRequest(t, req, http.StatusOK)
		} else {
			resp = MakeRequest(t, req, http.StatusOK)
		}
		doc := NewHTMLParser(t, resp.Body)

		flagsLinkCount := doc.Find(fmt.Sprintf(`a[href="%s/flags"]`, repoURL)).Length()
		assert.Equal(t, expectedLinkCount, flagsLinkCount)

		flagCount := doc.Find(".ui.info.message .ui.label").Length()
		assert.Equal(t, expectedFlagCount, flagCount)

		req = NewRequest(t, "GET", fmt.Sprintf("%s/flags", repoURL))
		if user != "" {
			session.MakeRequest(t, req, expectedStatus)
		} else {
			MakeRequest(t, req, expectedStatus)
		}
	}

	// Ensures that given a repo owner and a repo:
	// - An instance admin has access to flags, and sees the list on the repo home
	// - A repo admin does not have access to either, and does not see the list
	// - A passer by has no access to either, and does not see the list
	runTests := func(t *testing.T, ownerUser, repoURL string, expectedFlagCount int) {
		t.Run("as instance admin", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			assertFlagAccessAndCount(t, adminUser, repoURL, true, expectedFlagCount)
		})
		t.Run("as owner", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			assertFlagAccessAndCount(t, ownerUser, repoURL, false, 0)
		})
		t.Run("as other user", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			assertFlagAccessAndCount(t, otherUser, repoURL, false, 0)
		})
		t.Run("as non-logged in user", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			assertFlagAccessAndCount(t, "", repoURL, false, 0)
		})
	}

	// **************************
	// ** The tests themselves **
	// **************************
	t.Run("unflagged repo", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		defer ensureFlags(unflaggedRepo, []string{})()

		runTests(t, unflaggedOwner, unflaggedRepoURLStr, 0)
	})

	t.Run("flagged repo", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		defer ensureFlags(flaggedRepo, []string{"test-flag"})()

		runTests(t, flaggedOwner, flaggedRepoURLStr, 1)
	})

	t.Run("modifying flags", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		session := loginUser(t, adminUser)
		flaggedRepoManageURL := fmt.Sprintf("%s/flags", flaggedRepoURLStr)
		unflaggedRepoManageURL := fmt.Sprintf("%s/flags", unflaggedRepoURLStr)

		assertUIFlagStates := func(t *testing.T, url string, flagStates map[string]bool) {
			t.Helper()

			req := NewRequest(t, "GET", url)
			resp := session.MakeRequest(t, req, http.StatusOK)

			doc := NewHTMLParser(t, resp.Body)
			flagBoxes := doc.Find(`input[name="flags"]`)
			assert.Equal(t, len(flagStates), flagBoxes.Length())

			for name, state := range flagStates {
				_, checked := doc.Find(fmt.Sprintf(`input[value="%s"]`, name)).Attr("checked")
				assert.Equal(t, state, checked)
			}
		}

		t.Run("flag presence on the UI", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			defer ensureFlags(flaggedRepo, []string{"test-flag"})()

			assertUIFlagStates(t, flaggedRepoManageURL, map[string]bool{"test-flag": true})
		})

		t.Run("setting.Repository.SettableFlags is respected", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			defer test.MockVariableValue(&setting.Repository.SettableFlags, []string{"featured", "no-license"})()
			defer ensureFlags(flaggedRepo, []string{"test-flag"})()

			assertUIFlagStates(t, flaggedRepoManageURL, map[string]bool{
				"test-flag":  true,
				"featured":   false,
				"no-license": false,
			})
		})

		t.Run("removing flags", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			defer ensureFlags(flaggedRepo, []string{"test-flag"})()

			flagged := flaggedRepo.IsFlagged(db.DefaultContext)
			assert.True(t, flagged)

			req := NewRequestWithValues(t, "POST", flaggedRepoManageURL, map[string]string{
				"_csrf": GetCSRF(t, session, flaggedRepoManageURL),
			})
			session.MakeRequest(t, req, http.StatusSeeOther)

			flagged = flaggedRepo.IsFlagged(db.DefaultContext)
			assert.False(t, flagged)

			assertUIFlagStates(t, flaggedRepoManageURL, map[string]bool{})
		})

		t.Run("adding flags", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			defer ensureFlags(unflaggedRepo, []string{})()

			flagged := unflaggedRepo.IsFlagged(db.DefaultContext)
			assert.False(t, flagged)

			req := NewRequestWithValues(t, "POST", unflaggedRepoManageURL, map[string]string{
				"_csrf": GetCSRF(t, session, unflaggedRepoManageURL),
				"flags": "test-flag",
			})
			session.MakeRequest(t, req, http.StatusSeeOther)

			assertUIFlagStates(t, unflaggedRepoManageURL, map[string]bool{"test-flag": true})
		})
	})
}
