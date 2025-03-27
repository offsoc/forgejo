// Copyright 2023 The Gitea Authors. All rights reserved.
// Copyright 2024 The Forgejo Authors c/o Codeberg e.V.. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	actions_model "forgejo.org/models/actions"
	auth_model "forgejo.org/models/auth"
	repo_model "forgejo.org/models/repo"
	unit_model "forgejo.org/models/unit"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/git"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/test"
	"forgejo.org/routers"
	"forgejo.org/services/release"
	files_service "forgejo.org/services/repository/files"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBadges(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		prep := func(t *testing.T) (*repo_model.Repository, func()) {
			owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

			repo, _, f := tests.CreateDeclarativeRepo(t, owner, "",
				[]unit_model.Type{unit_model.TypeActions},
				[]unit_model.Type{unit_model.TypeIssues, unit_model.TypePullRequests, unit_model.TypeReleases},
				[]*files_service.ChangeRepoFile{
					{
						Operation:     "create",
						TreePath:      ".gitea/workflows/pr.yml",
						ContentReader: strings.NewReader("name: pr\non:\n  push:\njobs:\n  test:\n    runs-on: ubuntu-latest\n    steps:\n      - run: echo helloworld\n"),
					},
					{
						Operation:     "create",
						TreePath:      ".gitea/workflows/self-test.yaml",
						ContentReader: strings.NewReader("name: self-test\non:\n  push:\njobs:\n  test:\n    runs-on: ubuntu-latest\n    steps:\n      - run: echo helloworld\n"),
					},
					{
						Operation:     "create",
						TreePath:      ".gitea/workflows/tag-test.yaml",
						ContentReader: strings.NewReader("name: tags\non:\n  push:\n    tags: '*'\njobs:\n  test:\n    runs-on: ubuntu-latest\n    steps:\n      - run: echo helloworld\n"),
					},
				},
			)
			assert.Equal(t, 2, unittest.GetCount(t, &actions_model.ActionRun{RepoID: repo.ID}))

			return repo, f
		}

		assertBadge := func(t *testing.T, resp *httptest.ResponseRecorder, badge string) {
			t.Helper()

			assert.Equal(t, fmt.Sprintf("https://img.shields.io/badge/%s", badge), test.RedirectURL(resp))
		}

		t.Run("Workflows", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			repo, f := prep(t)
			defer f()

			// Actions disabled
			req := NewRequest(t, "GET", "/user2/repo1/badges/workflows/test.yaml/badge.svg")
			resp := MakeRequest(t, req, http.StatusSeeOther)
			assertBadge(t, resp, "test.yaml-Not%20found-crimson")

			req = NewRequest(t, "GET", "/user2/repo1/badges/workflows/test.yaml/badge.svg?branch=no-such-branch")
			resp = MakeRequest(t, req, http.StatusSeeOther)
			assertBadge(t, resp, "test.yaml-Not%20found-crimson")

			// Actions enabled
			req = NewRequestf(t, "GET", "/user2/%s/badges/workflows/pr.yml/badge.svg", repo.Name)
			resp = MakeRequest(t, req, http.StatusSeeOther)
			assertBadge(t, resp, "pr.yml-waiting-lightgrey")

			req = NewRequestf(t, "GET", "/user2/%s/badges/workflows/pr.yml/badge.svg?branch=main", repo.Name)
			resp = MakeRequest(t, req, http.StatusSeeOther)
			assertBadge(t, resp, "pr.yml-waiting-lightgrey")

			req = NewRequestf(t, "GET", "/user2/%s/badges/workflows/pr.yml/badge.svg?branch=no-such-branch", repo.Name)
			resp = MakeRequest(t, req, http.StatusSeeOther)
			assertBadge(t, resp, "pr.yml-Not%20found-crimson")

			req = NewRequestf(t, "GET", "/user2/%s/badges/workflows/pr.yml/badge.svg?event=cron", repo.Name)
			resp = MakeRequest(t, req, http.StatusSeeOther)
			assertBadge(t, resp, "pr.yml-Not%20found-crimson")

			// Workflow with a dash in its name
			req = NewRequestf(t, "GET", "/user2/%s/badges/workflows/self-test.yaml/badge.svg", repo.Name)
			resp = MakeRequest(t, req, http.StatusSeeOther)
			assertBadge(t, resp, "self--test.yaml-waiting-lightgrey")

			// GitHub compatibility
			req = NewRequestf(t, "GET", "/user2/%s/actions/workflows/pr.yml/badge.svg", repo.Name)
			resp = MakeRequest(t, req, http.StatusSeeOther)
			assertBadge(t, resp, "pr.yml-waiting-lightgrey")

			req = NewRequestf(t, "GET", "/user2/%s/actions/workflows/pr.yml/badge.svg?branch=main", repo.Name)
			resp = MakeRequest(t, req, http.StatusSeeOther)
			assertBadge(t, resp, "pr.yml-waiting-lightgrey")

			req = NewRequestf(t, "GET", "/user2/%s/actions/workflows/pr.yml/badge.svg?branch=no-such-branch", repo.Name)
			resp = MakeRequest(t, req, http.StatusSeeOther)
			assertBadge(t, resp, "pr.yml-Not%20found-crimson")

			req = NewRequestf(t, "GET", "/user2/%s/actions/workflows/pr.yml/badge.svg?event=cron", repo.Name)
			resp = MakeRequest(t, req, http.StatusSeeOther)
			assertBadge(t, resp, "pr.yml-Not%20found-crimson")

			t.Run("tagged", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				// With no tags, the workflow has no runs, and isn't found
				req := NewRequestf(t, "GET", "/user2/%s/actions/workflows/tag-test.yaml/badge.svg", repo.Name)
				resp := MakeRequest(t, req, http.StatusSeeOther)
				assertBadge(t, resp, "tag--test.yaml-Not%20found-crimson")

				// Lets create a tag!
				owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
				err := release.CreateNewTag(git.DefaultContext, owner, repo, "main", "v1", "message")
				require.NoError(t, err)

				// Now the workflow is waiting
				req = NewRequestf(t, "GET", "/user2/%s/actions/workflows/tag-test.yaml/badge.svg", repo.Name)
				resp = MakeRequest(t, req, http.StatusSeeOther)
				assertBadge(t, resp, "tag--test.yaml-waiting-lightgrey")
			})
		})

		t.Run("Stars", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequest(t, "GET", "/user2/repo1/badges/stars.svg")
			resp := MakeRequest(t, req, http.StatusSeeOther)

			assertBadge(t, resp, "stars-0-blue")

			t.Run("disabled stars", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()
				defer test.MockVariableValue(&setting.Repository.DisableStars, true)()
				defer test.MockVariableValue(&testWebRoutes, routers.NormalRoutes())()

				MakeRequest(t, req, http.StatusNotFound)
			})
		})

		t.Run("Issues", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			repo, f := prep(t)
			defer f()

			// Issues enabled
			req := NewRequest(t, "GET", "/user2/repo1/badges/issues.svg")
			resp := MakeRequest(t, req, http.StatusSeeOther)
			assertBadge(t, resp, "issues-2-blue")

			req = NewRequest(t, "GET", "/user2/repo1/badges/issues/open.svg")
			resp = MakeRequest(t, req, http.StatusSeeOther)
			assertBadge(t, resp, "issues-1%20open-blue")

			req = NewRequest(t, "GET", "/user2/repo1/badges/issues/closed.svg")
			resp = MakeRequest(t, req, http.StatusSeeOther)
			assertBadge(t, resp, "issues-1%20closed-blue")

			// Issues disabled
			req = NewRequestf(t, "GET", "/user2/%s/badges/issues.svg", repo.Name)
			resp = MakeRequest(t, req, http.StatusSeeOther)
			assertBadge(t, resp, "issues-Not%20found-crimson")

			req = NewRequestf(t, "GET", "/user2/%s/badges/issues/open.svg", repo.Name)
			resp = MakeRequest(t, req, http.StatusSeeOther)
			assertBadge(t, resp, "issues-Not%20found-crimson")

			req = NewRequestf(t, "GET", "/user2/%s/badges/issues/closed.svg", repo.Name)
			resp = MakeRequest(t, req, http.StatusSeeOther)
			assertBadge(t, resp, "issues-Not%20found-crimson")
		})

		t.Run("Pulls", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			repo, f := prep(t)
			defer f()

			// Pull requests enabled
			req := NewRequest(t, "GET", "/user2/repo1/badges/pulls.svg")
			resp := MakeRequest(t, req, http.StatusSeeOther)
			assertBadge(t, resp, "pulls-3-blue")

			req = NewRequest(t, "GET", "/user2/repo1/badges/pulls/open.svg")
			resp = MakeRequest(t, req, http.StatusSeeOther)
			assertBadge(t, resp, "pulls-3%20open-blue")

			req = NewRequest(t, "GET", "/user2/repo1/badges/pulls/closed.svg")
			resp = MakeRequest(t, req, http.StatusSeeOther)
			assertBadge(t, resp, "pulls-0%20closed-blue")

			// Pull requests disabled
			req = NewRequestf(t, "GET", "/user2/%s/badges/pulls.svg", repo.Name)
			resp = MakeRequest(t, req, http.StatusSeeOther)
			assertBadge(t, resp, "pulls-Not%20found-crimson")

			req = NewRequestf(t, "GET", "/user2/%s/badges/pulls/open.svg", repo.Name)
			resp = MakeRequest(t, req, http.StatusSeeOther)
			assertBadge(t, resp, "pulls-Not%20found-crimson")

			req = NewRequestf(t, "GET", "/user2/%s/badges/pulls/closed.svg", repo.Name)
			resp = MakeRequest(t, req, http.StatusSeeOther)
			assertBadge(t, resp, "pulls-Not%20found-crimson")
		})

		t.Run("Release", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			repo, f := prep(t)
			defer f()

			req := NewRequest(t, "GET", "/user2/repo1/badges/release.svg")
			resp := MakeRequest(t, req, http.StatusSeeOther)
			assertBadge(t, resp, "release-v1.1-blue")

			req = NewRequestf(t, "GET", "/user2/%s/badges/release.svg", repo.Name)
			resp = MakeRequest(t, req, http.StatusSeeOther)
			assertBadge(t, resp, "release-Not%20found-crimson")

			t.Run("Dashes in the name", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				session := loginUser(t, repo.Owner.Name)
				token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository)
				err := release.CreateNewTag(git.DefaultContext, repo.Owner, repo, "main", "repo-name-2.0", "dash in the tag name")
				require.NoError(t, err)
				createNewReleaseUsingAPI(t, token, repo.Owner, repo, "repo-name-2.0", "main", "dashed release", "dashed release")

				req := NewRequestf(t, "GET", "/user2/%s/badges/release.svg", repo.Name)
				resp := MakeRequest(t, req, http.StatusSeeOther)
				assertBadge(t, resp, "release-repo--name--2.0-blue")
			})
		})
	})
}
