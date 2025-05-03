// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	auth_model "forgejo.org/models/auth"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	api "forgejo.org/modules/structs"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func syncForkTest(t *testing.T, forkName, branchName string, webSync bool) {
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 20})

	baseRepo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	baseUser := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: baseRepo.OwnerID})

	session := loginUser(t, user.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository)

	// Create a new fork
	req := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/repos/%s/forks", baseRepo.FullName()), &api.CreateForkOption{Name: &forkName}).AddTokenAuth(token)
	MakeRequest(t, req, http.StatusAccepted)

	req = NewRequestf(t, "GET", "/api/v1/repos/%s/%s/sync_fork/%s", user.Name, forkName, branchName).AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusOK)

	var syncForkInfo *api.SyncForkInfo
	DecodeJSON(t, resp, &syncForkInfo)

	// This is a new fork, so the commits in both branches should be the same
	assert.False(t, syncForkInfo.Allowed)
	assert.Equal(t, syncForkInfo.BaseCommit, syncForkInfo.ForkCommit)

	// Make a commit on the base branch
	err := createOrReplaceFileInBranch(baseUser, baseRepo, "sync_fork.txt", branchName, "Hello")
	require.NoError(t, err)

	req = NewRequestf(t, "GET", "/api/v1/repos/%s/%s/sync_fork/%s", user.Name, forkName, branchName).AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)

	DecodeJSON(t, resp, &syncForkInfo)

	// The commits should no longer be the same and we can sync
	assert.True(t, syncForkInfo.Allowed)
	assert.NotEqual(t, syncForkInfo.BaseCommit, syncForkInfo.ForkCommit)

	// Sync the fork
	if webSync {
		session.MakeRequest(t, NewRequestWithValues(t, "POST", fmt.Sprintf("/%s/%s/sync_fork", user.Name, forkName), map[string]string{
			"_csrf":  GetCSRF(t, session, fmt.Sprintf("/%s/%s", user.Name, forkName)),
			"branch": branchName,
		}), http.StatusSeeOther)
	} else {
		req = NewRequestf(t, "POST", "/api/v1/repos/%s/%s/sync_fork/%s", user.Name, forkName, branchName).AddTokenAuth(token)
		MakeRequest(t, req, http.StatusNoContent)
	}

	req = NewRequestf(t, "GET", "/api/v1/repos/%s/%s/sync_fork/%s", user.Name, forkName, branchName).AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)

	DecodeJSON(t, resp, &syncForkInfo)

	// After the sync both commits should be the same again
	assert.False(t, syncForkInfo.Allowed)
	assert.Equal(t, syncForkInfo.BaseCommit, syncForkInfo.ForkCommit)
}

func TestAPIRepoSyncForkDefault(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		syncForkTest(t, "SyncForkDefault", "master", false)
	})
}

func TestAPIRepoSyncForkBranch(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		syncForkTest(t, "SyncForkBranch", "master", false)
	})
}

func TestWebRepoSyncForkBranch(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		syncForkTest(t, "SyncForkBranch", "master", true)
	})
}

func TestWebRepoSyncForkHomepage(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		baseRepo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
		baseOwner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: baseRepo.OwnerID})
		baseOwnerSession := loginUser(t, baseOwner.Name)

		forkOwner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 20})
		forkOwnerSession := loginUser(t, forkOwner.Name)
		token := getTokenForLoggedInUser(t, forkOwnerSession, auth_model.AccessTokenScopeWriteRepository)

		forkName := "SyncForkHomepage"
		forkLink := fmt.Sprintf("/%s/%s", forkOwner.Name, forkName)
		branchName := "<script>alert('0ko')</script>&amp;"
		branchHTMLEscaped := "&lt;script&gt;alert(&#39;0ko&#39;)&lt;/script&gt;&amp;amp;"
		branchURLEscaped := "%3Cscript%3Ealert%28%270ko%27%29%3C/script%3E&amp;amp%3B"

		// Rename branch "master" to test name escaping in the UI
		baseOwnerSession.MakeRequest(t, NewRequestWithValues(t, "POST",
			"/user2/repo1/settings/rename_branch", map[string]string{
				"_csrf": GetCSRF(t, baseOwnerSession, "/user2/repo1/branches"),
				"from":  "master",
				"to":    branchName,
			}), http.StatusSeeOther)

		// Create a new fork
		req := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/repos/%s/forks", baseRepo.FullName()), &api.CreateForkOption{Name: &forkName}).AddTokenAuth(token)
		MakeRequest(t, req, http.StatusAccepted)

		// Make a commit on the base branch
		err := createOrReplaceFileInBranch(baseOwner, baseRepo, "sync_fork.txt", branchName, "Hello")
		require.NoError(t, err)

		doc := NewHTMLParser(t, forkOwnerSession.MakeRequest(t,
			NewRequest(t, "GET", forkLink), http.StatusOK).Body)

		// Verify correct URL escaping of branch name in the form
		form := doc.Find("#sync_fork_msg form")
		assert.Equal(t, 1, form.Length())
		updateLink, exists := form.Attr("action")
		assert.True(t, exists)

		// Verify correct escaping of branch name in the message
		raw, _ := doc.Find("#sync_fork_msg").Html()
		assert.Contains(t, raw, fmt.Sprintf(`This branch is 1 commit behind <a href="http://localhost:%s/user2/repo1/src/branch/%s">user2/repo1:%s</a>`,
			u.Port(), branchURLEscaped, branchHTMLEscaped))

		// Verify that the form link doesn't do anything for a GET request
		forkOwnerSession.MakeRequest(t, NewRequest(t, "GET", updateLink), http.StatusMethodNotAllowed)

		// Verify that the form link does not error out
		forkOwnerSession.MakeRequest(t, NewRequestWithValues(t, "POST", updateLink, map[string]string{
			"_csrf":  GetCSRF(t, forkOwnerSession, forkLink),
			"branch": branchName,
		}), http.StatusSeeOther)
	})
}
