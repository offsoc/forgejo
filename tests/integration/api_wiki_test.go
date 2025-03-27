// Copyright 2021 The Gitea Authors. All rights reserved.
// Copyright 2024 The Forgejo Authors c/o Codeberg e.V.. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	auth_model "forgejo.org/models/auth"
	repo_model "forgejo.org/models/repo"
	unit_model "forgejo.org/models/unit"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/optional"
	api "forgejo.org/modules/structs"
	repo_service "forgejo.org/services/repository"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIRenameWikiBranch(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	username := "user2"
	session := loginUser(t, username)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository)

	repoURLStr := fmt.Sprintf("/api/v1/repos/%s/%s", username, "repo1")
	wikiBranch := "wiki"
	req := NewRequestWithJSON(t, "PATCH", repoURLStr, &api.EditRepoOption{
		WikiBranch: &wikiBranch,
	}).AddTokenAuth(token)
	MakeRequest(t, req, http.StatusOK)

	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	assert.Equal(t, "wiki", repo.WikiBranch)

	req = NewRequest(t, "GET", repoURLStr)
	resp := MakeRequest(t, req, http.StatusOK)
	var repoData *api.Repository
	DecodeJSON(t, resp, &repoData)
	assert.Equal(t, "wiki", repoData.WikiBranch)
}

func TestAPIGetWikiPage(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	username := "user2"

	urlStr := fmt.Sprintf("/api/v1/repos/%s/%s/wiki/page/Home", username, "repo1")

	req := NewRequest(t, "GET", urlStr)
	resp := MakeRequest(t, req, http.StatusOK)
	var page *api.WikiPage
	DecodeJSON(t, resp, &page)

	assert.Equal(t, &api.WikiPage{
		WikiPageMetaData: &api.WikiPageMetaData{
			Title:   "Home",
			HTMLURL: page.HTMLURL,
			SubURL:  "Home",
			LastCommit: &api.WikiCommit{
				ID: "2c54faec6c45d31c1abfaecdab471eac6633738a",
				Author: &api.CommitUser{
					Identity: api.Identity{
						Name:  "Ethan Koenig",
						Email: "ethantkoenig@gmail.com",
					},
					Date: "2017-11-27T04:31:18Z",
				},
				Committer: &api.CommitUser{
					Identity: api.Identity{
						Name:  "Ethan Koenig",
						Email: "ethantkoenig@gmail.com",
					},
					Date: "2017-11-27T04:31:18Z",
				},
				Message: "Add Home.md\n",
			},
		},
		ContentBase64: base64.RawStdEncoding.EncodeToString(
			[]byte("# Home page\n\nThis is the home page!\n"),
		),
		CommitCount: 1,
		Sidebar:     "",
		Footer:      "",
	}, page)
}

func TestAPIListWikiPages(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	username := "user2"

	urlStr := fmt.Sprintf("/api/v1/repos/%s/%s/wiki/pages", username, "repo1")

	req := NewRequest(t, "GET", urlStr)
	resp := MakeRequest(t, req, http.StatusOK)

	var meta []*api.WikiPageMetaData
	DecodeJSON(t, resp, &meta)

	dummymeta := []*api.WikiPageMetaData{
		{
			Title:   "Home",
			HTMLURL: meta[0].HTMLURL,
			SubURL:  "Home",
			LastCommit: &api.WikiCommit{
				ID: "2c54faec6c45d31c1abfaecdab471eac6633738a",
				Author: &api.CommitUser{
					Identity: api.Identity{
						Name:  "Ethan Koenig",
						Email: "ethantkoenig@gmail.com",
					},
					Date: "2017-11-27T04:31:18Z",
				},
				Committer: &api.CommitUser{
					Identity: api.Identity{
						Name:  "Ethan Koenig",
						Email: "ethantkoenig@gmail.com",
					},
					Date: "2017-11-27T04:31:18Z",
				},
				Message: "Add Home.md\n",
			},
		},
		{
			Title:   "Long Page",
			HTMLURL: meta[1].HTMLURL,
			SubURL:  "Long-Page",
			LastCommit: &api.WikiCommit{
				ID: "d49ac742d44063dcf69d4e0afe725813b777dd89",
				Author: &api.CommitUser{
					Identity: api.Identity{
						Name:  "Oto Šťáva",
						Email: "oto.stava@gmail.com",
					},
					Date: "2024-11-23T11:16:51Z",
				},
				Committer: &api.CommitUser{
					Identity: api.Identity{
						Name:  "Oto Šťáva",
						Email: "oto.stava@gmail.com",
					},
					Date: "2024-11-23T11:16:51Z",
				},
				Message: "add long page\n",
			},
		},
		{
			Title:   "Page With Image",
			HTMLURL: meta[2].HTMLURL,
			SubURL:  "Page-With-Image",
			LastCommit: &api.WikiCommit{
				ID: "0cf15c3f66ec8384480ed9c3cf87c9e97fbb0ec3",
				Author: &api.CommitUser{
					Identity: api.Identity{
						Name:  "Gabriel Silva Simões",
						Email: "simoes.sgabriel@gmail.com",
					},
					Date: "2019-01-25T01:41:55Z",
				},
				Committer: &api.CommitUser{
					Identity: api.Identity{
						Name:  "Gabriel Silva Simões",
						Email: "simoes.sgabriel@gmail.com",
					},
					Date: "2019-01-25T01:41:55Z",
				},
				Message: "Add jpeg.jpg and page with image\n",
			},
		},
		{
			Title:   "Page With Spaced Name",
			HTMLURL: meta[3].HTMLURL,
			SubURL:  "Page-With-Spaced-Name",
			LastCommit: &api.WikiCommit{
				ID: "c10d10b7e655b3dab1f53176db57c8219a5488d6",
				Author: &api.CommitUser{
					Identity: api.Identity{
						Name:  "Gabriel Silva Simões",
						Email: "simoes.sgabriel@gmail.com",
					},
					Date: "2019-01-25T01:39:51Z",
				},
				Committer: &api.CommitUser{
					Identity: api.Identity{
						Name:  "Gabriel Silva Simões",
						Email: "simoes.sgabriel@gmail.com",
					},
					Date: "2019-01-25T01:39:51Z",
				},
				Message: "Add page with spaced name\n",
			},
		},
		{
			Title:   "Unescaped File",
			HTMLURL: meta[4].HTMLURL,
			SubURL:  "Unescaped-File",
			LastCommit: &api.WikiCommit{
				ID: "0dca5bd9b5d7ef937710e056f575e86c0184ba85",
				Author: &api.CommitUser{
					Identity: api.Identity{
						Name:  "6543",
						Email: "6543@obermui.de",
					},
					Date: "2021-07-19T16:42:46Z",
				},
				Committer: &api.CommitUser{
					Identity: api.Identity{
						Name:  "6543",
						Email: "6543@obermui.de",
					},
					Date: "2021-07-19T16:42:46Z",
				},
				Message: "add unescaped file\n",
			},
		},
		{
			Title:   "XSS",
			HTMLURL: meta[5].HTMLURL,
			SubURL:  "XSS",
			LastCommit: &api.WikiCommit{
				ID: "f54f5a6b7c4f83b606600e43186165854f189530",
				Author: &api.CommitUser{
					Identity: api.Identity{
						Name:  "Gusted<script class=\"evil\">alert('Oh no!');</script>",
						Email: "valid@example.org",
					},
					Date: "2024-01-31T00:00:00Z",
				},
				Committer: &api.CommitUser{
					Identity: api.Identity{
						Name:  "Gusted<script class=\"evil\">alert('Oh no!');</script>",
						Email: "valid@example.org",
					},
					Date: "2024-01-31T00:00:00Z",
				},
				Message: "Yay XSS",
			},
		},
	}

	assert.Equal(t, dummymeta, meta)
}

func TestAPINewWikiPage(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	for _, title := range []string{
		"New page",
		"&&&&",
	} {
		username := "user2"
		session := loginUser(t, username)
		token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository)

		urlStr := fmt.Sprintf("/api/v1/repos/%s/%s/wiki/new", username, "repo1")

		req := NewRequestWithJSON(t, "POST", urlStr, &api.CreateWikiPageOptions{
			Title:         title,
			ContentBase64: base64.StdEncoding.EncodeToString([]byte("Wiki page content for API unit tests")),
			Message:       "",
		}).AddTokenAuth(token)
		MakeRequest(t, req, http.StatusCreated)
	}
}

func TestAPIEditWikiPage(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	username := "user2"
	session := loginUser(t, username)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository)

	urlStr := fmt.Sprintf("/api/v1/repos/%s/%s/wiki/page/Page-With-Spaced-Name", username, "repo1")

	req := NewRequestWithJSON(t, "PATCH", urlStr, &api.CreateWikiPageOptions{
		Title:         "edited title",
		ContentBase64: base64.StdEncoding.EncodeToString([]byte("Edited wiki page content for API unit tests")),
		Message:       "",
	}).AddTokenAuth(token)
	MakeRequest(t, req, http.StatusOK)
}

func TestAPIEditOtherWikiPage(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// (drive-by-user) user, session, and token for a drive-by wiki editor
	username := "drive-by-user"
	req := NewRequestWithValues(t, "POST", "/user/sign_up", map[string]string{
		"user_name": username,
		"email":     "drive-by@example.com",
		"password":  "examplePassword!1",
		"retype":    "examplePassword!1",
	})
	MakeRequest(t, req, http.StatusSeeOther)
	session := loginUserWithPassword(t, username, "examplePassword!1")
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository)

	// (user2) user for the user whose wiki we're going to edit (as drive-by-user)
	otherUsername := "user2"

	// Creating a new Wiki page on user2's repo as user1 fails
	testCreateWiki := func(expectedStatusCode int) {
		urlStr := fmt.Sprintf("/api/v1/repos/%s/%s/wiki/new", otherUsername, "repo1")
		req := NewRequestWithJSON(t, "POST", urlStr, &api.CreateWikiPageOptions{
			Title:         "Globally Edited Page",
			ContentBase64: base64.StdEncoding.EncodeToString([]byte("Wiki page content for API unit tests")),
			Message:       "",
		}).AddTokenAuth(token)
		session.MakeRequest(t, req, expectedStatusCode)
	}
	testCreateWiki(http.StatusForbidden)

	// Update the repo settings for user2's repo to enable globally writeable wiki
	ctx := t.Context()
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	var units []repo_model.RepoUnit
	units = append(units, repo_model.RepoUnit{
		RepoID:             repo.ID,
		Type:               unit_model.TypeWiki,
		Config:             new(repo_model.UnitConfig),
		DefaultPermissions: repo_model.UnitAccessModeWrite,
	})
	err := repo_service.UpdateRepositoryUnits(ctx, repo, units, nil)
	require.NoError(t, err)

	// Creating a new Wiki page on user2's repo works now
	testCreateWiki(http.StatusCreated)
}

func TestAPISetWikiGlobalEditability(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: "user2"})
	session := loginUser(t, user.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository)

	// Create a new repository for testing purposes
	repo, _, f := tests.CreateDeclarativeRepo(t, user, "", []unit_model.Type{
		unit_model.TypeCode,
		unit_model.TypeWiki,
	}, nil, nil)
	defer f()
	urlStr := fmt.Sprintf("/api/v1/repos/%s", repo.FullName())

	assertGlobalEditability := func(t *testing.T, editability bool) {
		t.Helper()

		req := NewRequest(t, "GET", urlStr)
		resp := MakeRequest(t, req, http.StatusOK)

		var opts api.Repository
		DecodeJSON(t, resp, &opts)

		assert.Equal(t, opts.GloballyEditableWiki, editability)
	}

	t.Run("api includes GloballyEditableWiki", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		assertGlobalEditability(t, false)
	})

	t.Run("api can turn on GloballyEditableWiki", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		globallyEditable := true
		req := NewRequestWithJSON(t, "PATCH", urlStr, &api.EditRepoOption{
			GloballyEditableWiki: &globallyEditable,
		}).AddTokenAuth(token)
		MakeRequest(t, req, http.StatusOK)

		assertGlobalEditability(t, true)
	})

	t.Run("disabling the wiki disables GloballyEditableWiki", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		hasWiki := false
		req := NewRequestWithJSON(t, "PATCH", urlStr, &api.EditRepoOption{
			HasWiki: &hasWiki,
		}).AddTokenAuth(token)
		MakeRequest(t, req, http.StatusOK)

		assertGlobalEditability(t, false)
	})
}

func TestAPIListPageRevisions(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	username := "user2"

	urlStr := fmt.Sprintf("/api/v1/repos/%s/%s/wiki/revisions/Home", username, "repo1")

	req := NewRequest(t, "GET", urlStr)
	resp := MakeRequest(t, req, http.StatusOK)

	var revisions *api.WikiCommitList
	DecodeJSON(t, resp, &revisions)

	dummyrevisions := &api.WikiCommitList{
		WikiCommits: []*api.WikiCommit{
			{
				ID: "2c54faec6c45d31c1abfaecdab471eac6633738a",
				Author: &api.CommitUser{
					Identity: api.Identity{
						Name:  "Ethan Koenig",
						Email: "ethantkoenig@gmail.com",
					},
					Date: "2017-11-27T04:31:18Z",
				},
				Committer: &api.CommitUser{
					Identity: api.Identity{
						Name:  "Ethan Koenig",
						Email: "ethantkoenig@gmail.com",
					},
					Date: "2017-11-27T04:31:18Z",
				},
				Message: "Add Home.md\n",
			},
		},
		Count: 1,
	}

	assert.Equal(t, dummyrevisions, revisions)
}

func TestAPIWikiNonMasterBranch(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, _ *url.URL) {
		user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
		repo, _, f := tests.CreateDeclarativeRepoWithOptions(t, user, tests.DeclarativeRepoOptions{
			WikiBranch: optional.Some("main"),
		})
		defer f()

		uris := []string{
			"revisions/Home",
			"pages",
			"page/Home",
		}
		baseURL := fmt.Sprintf("/api/v1/repos/%s/wiki", repo.FullName())
		for _, uri := range uris {
			t.Run(uri, func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				req := NewRequestf(t, "GET", "%s/%s", baseURL, uri)
				MakeRequest(t, req, http.StatusOK)
			})
		}
	})
}
