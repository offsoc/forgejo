// Copyright 2023 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"testing"

	"forgejo.org/models/activities"
	"forgejo.org/models/db"
	issue_model "forgejo.org/models/issues"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/translation"
	forgejo_context "forgejo.org/services/context"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
)

func BlockUser(t *testing.T, doer, blockedUser *user_model.User) {
	t.Helper()

	unittest.AssertNotExistsBean(t, &user_model.BlockedUser{BlockID: blockedUser.ID, UserID: doer.ID})

	session := loginUser(t, doer.Name)
	req := NewRequestWithValues(t, "POST", "/"+blockedUser.Name, map[string]string{
		"_csrf":  GetCSRF(t, session, "/"+blockedUser.Name),
		"action": "block",
	})
	session.MakeRequest(t, req, http.StatusOK)

	assert.True(t, unittest.BeanExists(t, &user_model.BlockedUser{BlockID: blockedUser.ID, UserID: doer.ID}))
}

// TestBlockUser ensures that users can execute blocking related actions can
// happen under the correct conditions.
func TestBlockUser(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 8})
	blockedUser := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
	session := loginUser(t, doer.Name)

	t.Run("Block", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		BlockUser(t, doer, blockedUser)
	})

	// Unblock user.
	t.Run("Unblock", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		req := NewRequestWithValues(t, "POST", "/"+blockedUser.Name, map[string]string{
			"_csrf":  GetCSRF(t, session, "/"+blockedUser.Name),
			"action": "unblock",
		})
		session.MakeRequest(t, req, http.StatusOK)

		unittest.AssertNotExistsBean(t, &user_model.BlockedUser{BlockID: blockedUser.ID, UserID: doer.ID})
	})

	t.Run("Organization as target", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		targetOrg := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 3, Type: user_model.UserTypeOrganization})

		t.Run("Block", func(t *testing.T) {
			req := NewRequestWithValues(t, "POST", "/"+targetOrg.Name, map[string]string{
				"_csrf":  GetCSRF(t, session, "/"+targetOrg.Name),
				"action": "block",
			})
			resp := session.MakeRequest(t, req, http.StatusBadRequest)

			assert.Contains(t, resp.Body.String(), "Action \\\"block\\\" failed")
		})

		t.Run("Unblock", func(t *testing.T) {
			req := NewRequestWithValues(t, "POST", "/"+targetOrg.Name, map[string]string{
				"_csrf":  GetCSRF(t, session, "/"+targetOrg.Name),
				"action": "unblock",
			})
			resp := session.MakeRequest(t, req, http.StatusBadRequest)

			assert.Contains(t, resp.Body.String(), "Action \\\"unblock\\\" failed")
		})
	})
}

// TestBlockUserFromOrganization ensures that an organisation can block and unblock an user.
func TestBlockUserFromOrganization(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 15})
	blockedUser := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
	org := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 17, Type: user_model.UserTypeOrganization})
	unittest.AssertNotExistsBean(t, &user_model.BlockedUser{BlockID: blockedUser.ID, UserID: org.ID})
	session := loginUser(t, doer.Name)

	t.Run("Block user", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequestWithValues(t, "POST", org.OrganisationLink()+"/settings/blocked_users/block", map[string]string{
			"_csrf": GetCSRF(t, session, org.OrganisationLink()+"/settings/blocked_users"),
			"uname": blockedUser.Name,
		})
		session.MakeRequest(t, req, http.StatusSeeOther)
		assert.True(t, unittest.BeanExists(t, &user_model.BlockedUser{BlockID: blockedUser.ID, UserID: org.ID}))
	})

	t.Run("Unblock user", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequestWithValues(t, "POST", org.OrganisationLink()+"/settings/blocked_users/unblock", map[string]string{
			"_csrf":   GetCSRF(t, session, org.OrganisationLink()+"/settings/blocked_users"),
			"user_id": strconv.FormatInt(blockedUser.ID, 10),
		})
		session.MakeRequest(t, req, http.StatusSeeOther)
		unittest.AssertNotExistsBean(t, &user_model.BlockedUser{BlockID: blockedUser.ID, UserID: org.ID})
	})

	t.Run("Organization as target", func(t *testing.T) {
		targetOrg := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 3, Type: user_model.UserTypeOrganization})

		t.Run("Block", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequestWithValues(t, "POST", org.OrganisationLink()+"/settings/blocked_users/block", map[string]string{
				"_csrf": GetCSRF(t, session, org.OrganisationLink()+"/settings/blocked_users"),
				"uname": targetOrg.Name,
			})
			session.MakeRequest(t, req, http.StatusInternalServerError)
			unittest.AssertNotExistsBean(t, &user_model.BlockedUser{BlockID: blockedUser.ID, UserID: targetOrg.ID})
		})

		t.Run("Unblock", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequestWithValues(t, "POST", org.OrganisationLink()+"/settings/blocked_users/unblock", map[string]string{
				"_csrf":   GetCSRF(t, session, org.OrganisationLink()+"/settings/blocked_users"),
				"user_id": strconv.FormatInt(targetOrg.ID, 10),
			})
			session.MakeRequest(t, req, http.StatusInternalServerError)
		})
	})

	t.Run("Block the doer", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequestWithValues(t, "POST", org.OrganisationLink()+"/settings/blocked_users/block", map[string]string{
			"_csrf": GetCSRF(t, session, org.OrganisationLink()+"/settings/blocked_users"),
			"uname": doer.Name,
		})
		session.MakeRequest(t, req, http.StatusSeeOther)
		assert.False(t, unittest.BeanExists(t, &user_model.BlockedUser{BlockID: doer.ID, UserID: org.ID}))
		flashCookie := session.GetCookie(forgejo_context.CookieNameFlash)
		assert.NotNil(t, flashCookie)
		assert.Equal(t, "error%3DYou%2Bcannot%2Bblock%2Byourself.", flashCookie.Value)
	})
}

// TestBlockActions ensures that certain actions cannot be performed as a doer
// and as a blocked user and are handled cleanly after the blocking has taken
// place.
func TestBlockActions(t *testing.T) {
	defer unittest.OverrideFixtures("tests/integration/fixtures/TestBlockActions")()
	defer tests.PrepareTestEnv(t)()

	doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	blockedUser := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
	blockedUser2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 10})
	repo1 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1, OwnerID: doer.ID})
	repo2 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 2, OwnerID: doer.ID})
	repo7 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 7, OwnerID: blockedUser2.ID})
	issue4 := unittest.AssertExistsAndLoadBean(t, &issue_model.Issue{ID: 4, RepoID: repo2.ID})
	issue4URL := fmt.Sprintf("/%s/issues/%d", repo2.FullName(), issue4.Index)
	repo42 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 42, OwnerID: doer.ID})
	issue10 := unittest.AssertExistsAndLoadBean(t, &issue_model.Issue{ID: 10, RepoID: repo42.ID}, unittest.Cond("poster_id != ?", doer.ID))
	issue10URL := fmt.Sprintf("/%s/issues/%d", repo42.FullName(), issue10.Index)
	// NOTE: Sessions shouldn't be shared, because in some situations flash
	// messages are persistent and that would interfere with accurate test
	// results.

	BlockUser(t, doer, blockedUser)
	BlockUser(t, doer, blockedUser2)

	type errorJSON struct {
		Error string `json:"errorMessage"`
	}
	locale := translation.NewLocale("en-US")

	// Ensures that issue creation on doer's owned repositories are blocked.
	t.Run("Issue creation", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		session := loginUser(t, blockedUser.Name)
		link := fmt.Sprintf("%s/issues/new", repo2.FullName())

		req := NewRequestWithValues(t, "POST", link, map[string]string{
			"_csrf":   GetCSRF(t, session, link),
			"title":   "Title",
			"content": "Hello!",
		})
		resp := session.MakeRequest(t, req, http.StatusBadRequest)

		var errorResp errorJSON
		DecodeJSON(t, resp, &errorResp)

		assert.EqualValues(t, locale.Tr("repo.issues.blocked_by_user"), errorResp.Error)
	})

	// Ensures that pull creation on doer's owned repositories are blocked.
	t.Run("Pull creation", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		session := loginUser(t, blockedUser.Name)
		link := fmt.Sprintf("%s/compare/v1.1...master", repo1.FullName())

		req := NewRequestWithValues(t, "POST", link, map[string]string{
			"_csrf":   GetCSRF(t, session, link),
			"title":   "Title",
			"content": "Hello!",
		})
		resp := session.MakeRequest(t, req, http.StatusBadRequest)

		var errorResp errorJSON
		DecodeJSON(t, resp, &errorResp)

		assert.EqualValues(t, locale.Tr("repo.pulls.blocked_by_user"), errorResp.Error)
	})

	// Ensures that comment creation on doer's owned repositories and doer's
	// posted issues are blocked.
	t.Run("Comment creation", func(t *testing.T) {
		expectedMessage := locale.Tr("repo.comment.blocked_by_user")

		t.Run("Blocked by repository owner", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			session := loginUser(t, blockedUser.Name)

			req := NewRequestWithValues(t, "POST", path.Join(issue10URL, "/comments"), map[string]string{
				"_csrf":   GetCSRF(t, session, issue10URL),
				"content": "Not a kind comment",
			})
			resp := session.MakeRequest(t, req, http.StatusBadRequest)

			var errorResp errorJSON
			DecodeJSON(t, resp, &errorResp)

			assert.EqualValues(t, expectedMessage, errorResp.Error)

			req = NewRequest(t, "GET", issue10URL)
			resp = session.MakeRequest(t, req, http.StatusOK)
			htmlDoc := NewHTMLParser(t, resp.Body)
			msg := htmlDoc.doc.Find("div .warning").Text()
			assert.Contains(t, msg, expectedMessage)
		})

		t.Run("Blocked by issue poster", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			repo5 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 5})
			issue15 := unittest.AssertExistsAndLoadBean(t, &issue_model.Issue{ID: 15, RepoID: repo5.ID, PosterID: doer.ID})

			session := loginUser(t, blockedUser.Name)
			issueURL := fmt.Sprintf("/%s/%s/issues/%d", url.PathEscape(repo5.OwnerName), url.PathEscape(repo5.Name), issue15.Index)

			req := NewRequestWithValues(t, "POST", path.Join(issueURL, "/comments"), map[string]string{
				"_csrf":   GetCSRF(t, session, issueURL),
				"content": "Not a kind comment",
			})
			resp := session.MakeRequest(t, req, http.StatusBadRequest)

			var errorResp errorJSON
			DecodeJSON(t, resp, &errorResp)

			assert.EqualValues(t, expectedMessage, errorResp.Error)

			req = NewRequest(t, "GET", issue10URL)
			resp = session.MakeRequest(t, req, http.StatusOK)
			htmlDoc := NewHTMLParser(t, resp.Body)
			msg := htmlDoc.doc.Find("div .warning").Text()
			assert.Contains(t, msg, expectedMessage)
		})
	})

	// Ensures that reactions on doer's owned issues and doer's owned comments are
	// blocked.
	t.Run("Add a reaction", func(t *testing.T) {
		type reactionResponse struct {
			Empty bool `json:"empty"`
		}

		t.Run("On a issue", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			session := loginUser(t, blockedUser.Name)

			req := NewRequestWithValues(t, "POST", path.Join(issue4URL, "/reactions/react"), map[string]string{
				"_csrf":   GetCSRF(t, session, issue4URL),
				"content": "eyes",
			})
			resp := session.MakeRequest(t, req, http.StatusOK)

			var respBody reactionResponse
			DecodeJSON(t, resp, &respBody)

			assert.True(t, respBody.Empty)
		})

		t.Run("On a comment", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			comment := unittest.AssertExistsAndLoadBean(t, &issue_model.Comment{ID: 1008, PosterID: doer.ID, IssueID: issue4.ID})

			session := loginUser(t, blockedUser.Name)

			req := NewRequestWithValues(t, "POST", fmt.Sprintf("%s/comments/%d/reactions/react", repo2.FullName(), comment.ID), map[string]string{
				"_csrf":   GetCSRF(t, session, issue4URL),
				"content": "eyes",
			})
			resp := session.MakeRequest(t, req, http.StatusOK)

			var respBody reactionResponse
			DecodeJSON(t, resp, &respBody)

			assert.True(t, respBody.Empty)
		})
	})

	// Ensures that the doer and blocked user cannot follow each other.
	t.Run("Follow", func(t *testing.T) {
		// Sanity checks to make sure doing these tests are valid.
		unittest.AssertNotExistsBean(t, &user_model.Follow{UserID: doer.ID, FollowID: blockedUser.ID})
		unittest.AssertNotExistsBean(t, &user_model.Follow{UserID: blockedUser.ID, FollowID: doer.ID})

		// Doer cannot follow blocked user.
		t.Run("Doer follow blocked user", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			session := loginUser(t, doer.Name)

			req := NewRequestWithValues(t, "POST", "/"+blockedUser.Name, map[string]string{
				"_csrf":  GetCSRF(t, session, "/"+blockedUser.Name),
				"action": "follow",
			})
			resp := session.MakeRequest(t, req, http.StatusOK)

			htmlDoc := NewHTMLParser(t, resp.Body)
			assert.Contains(t, htmlDoc.Find("#flash-message").Text(), "You cannot follow this user because you have blocked this user or this user has blocked you.")

			// Assert it still doesn't exist.
			unittest.AssertNotExistsBean(t, &user_model.Follow{UserID: doer.ID, FollowID: blockedUser.ID})
		})

		// Blocked user cannot follow doer.
		t.Run("Blocked user follow doer", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			session := loginUser(t, blockedUser.Name)

			req := NewRequestWithValues(t, "POST", "/"+doer.Name, map[string]string{
				"_csrf":  GetCSRF(t, session, "/"+doer.Name),
				"action": "follow",
			})
			resp := session.MakeRequest(t, req, http.StatusOK)

			htmlDoc := NewHTMLParser(t, resp.Body)
			assert.Contains(t, htmlDoc.Find("#flash-message").Text(), "You cannot follow this user because you have blocked this user or this user has blocked you.")

			unittest.AssertNotExistsBean(t, &user_model.Follow{UserID: blockedUser.ID, FollowID: doer.ID})
		})
	})

	// Ensures that the doer and blocked user cannot add each each other as collaborators.
	t.Run("Add collaborator", func(t *testing.T) {
		t.Run("Doer Add BlockedUser", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			session := loginUser(t, doer.Name)
			link := fmt.Sprintf("/%s/settings/collaboration", repo2.FullName())

			req := NewRequestWithValues(t, "POST", link, map[string]string{
				"_csrf":        GetCSRF(t, session, link),
				"collaborator": blockedUser2.Name,
			})
			session.MakeRequest(t, req, http.StatusSeeOther)

			flashCookie := session.GetCookie(forgejo_context.CookieNameFlash)
			assert.NotNil(t, flashCookie)
			assert.Equal(t, "error%3DCannot%2Badd%2Bthe%2Bcollaborator%252C%2Bbecause%2Bthe%2Brepository%2Bowner%2Bhas%2Bblocked%2Bthem.", flashCookie.Value)
		})

		t.Run("BlockedUser Add doer", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			session := loginUser(t, blockedUser2.Name)
			link := fmt.Sprintf("/%s/settings/collaboration", repo7.FullName())

			req := NewRequestWithValues(t, "POST", link, map[string]string{
				"_csrf":        GetCSRF(t, session, link),
				"collaborator": doer.Name,
			})
			session.MakeRequest(t, req, http.StatusSeeOther)

			flashCookie := session.GetCookie(forgejo_context.CookieNameFlash)
			assert.NotNil(t, flashCookie)
			assert.Equal(t, "error%3DCannot%2Badd%2Bthe%2Bcollaborator%252C%2Bbecause%2Bthey%2Bhave%2Bblocked%2Bthe%2Brepository%2Bowner.", flashCookie.Value)
		})
	})

	// Ensures that the blocked user cannot transfer a repository to the doer.
	t.Run("Repository transfer", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		session := loginUser(t, blockedUser2.Name)
		link := fmt.Sprintf("%s/settings", repo7.FullName())

		req := NewRequestWithValues(t, "POST", link, map[string]string{
			"_csrf":          GetCSRF(t, session, link),
			"action":         "transfer",
			"repo_name":      repo7.FullName(),
			"new_owner_name": doer.Name,
		})
		resp := session.MakeRequest(t, req, http.StatusOK)

		htmlDoc := NewHTMLParser(t, resp.Body)
		assert.Contains(t,
			htmlDoc.doc.Find(".ui.negative.message").Text(),
			translation.NewLocale("en-US").Tr("repo.settings.new_owner_blocked_doer"),
		)
	})
}

func TestBlockedNotification(t *testing.T) {
	defer unittest.OverrideFixtures("tests/integration/fixtures/TestBlockedNotifications")()
	defer tests.PrepareTestEnv(t)()

	doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	normalUser := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
	blockedUser := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 10})
	issue := unittest.AssertExistsAndLoadBean(t, &issue_model.Issue{ID: 1000})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: issue.RepoID})
	issueURL := fmt.Sprintf("%s/issues/%d", repo.FullName(), issue.Index)
	notificationBean := &activities.Notification{UserID: doer.ID, RepoID: repo.ID, IssueID: issue.ID}

	assert.False(t, user_model.IsBlocked(db.DefaultContext, doer.ID, normalUser.ID))
	BlockUser(t, doer, blockedUser)

	mentionDoer := func(t *testing.T, session *TestSession) {
		t.Helper()

		req := NewRequestWithValues(t, "POST", issueURL+"/comments", map[string]string{
			"_csrf":   GetCSRF(t, session, issueURL),
			"content": "I'm annoying. Pinging @" + doer.Name,
		})
		session.MakeRequest(t, req, http.StatusOK)
	}

	t.Run("Blocks notification of blocked user", func(t *testing.T) {
		session := loginUser(t, blockedUser.Name)

		unittest.AssertNotExistsBean(t, notificationBean)
		mentionDoer(t, session)
		unittest.AssertNotExistsBean(t, notificationBean)
	})

	t.Run("Do not block notifications of normal user", func(t *testing.T) {
		session := loginUser(t, normalUser.Name)

		unittest.AssertNotExistsBean(t, notificationBean)
		mentionDoer(t, session)
		unittest.AssertExistsAndLoadBean(t, notificationBean)
	})
}
