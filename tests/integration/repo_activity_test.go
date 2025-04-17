// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"testing"

	auth_model "forgejo.org/models/auth"
	"forgejo.org/models/db"
	repo_model "forgejo.org/models/repo"
	unit_model "forgejo.org/models/unit"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/test"
	repo_service "forgejo.org/services/repository"
	"forgejo.org/tests"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepoActivity(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, giteaURL *url.URL) {
		session := loginUser(t, "user1")

		// Create PRs (1 merged & 2 proposed)
		testRepoFork(t, session, "user2", "repo1", "user1", "repo1")
		testEditFile(t, session, "user1", "repo1", "master", "README.md", "Hello, World (Edited)\n")
		resp := testPullCreate(t, session, "user1", "repo1", false, "master", "master", "This is a pull title")
		elem := strings.Split(test.RedirectURL(resp), "/")
		assert.Equal(t, "pulls", elem[3])
		testPullMerge(t, session, elem[1], elem[2], elem[4], repo_model.MergeStyleMerge, false)

		testEditFileToNewBranch(t, session, "user1", "repo1", "master", "feat/better_readme", "README.md", "Hello, World (Edited Again)\n")
		testPullCreate(t, session, "user1", "repo1", false, "master", "feat/better_readme", "This is a pull title")

		testEditFileToNewBranch(t, session, "user1", "repo1", "master", "feat/much_better_readme", "README.md", "Hello, World (Edited More)\n")
		testPullCreate(t, session, "user1", "repo1", false, "master", "feat/much_better_readme", "This is a pull title")

		// Create issues (3 new issues)
		testNewIssue(t, session, "user2", "repo1", "Issue 1", "Description 1")
		testNewIssue(t, session, "user2", "repo1", "Issue 2", "Description 2")
		testNewIssue(t, session, "user2", "repo1", "Issue 3", "Description 3")

		// Create releases (1 release, 1 pre-release, 1 release-draft, 1 tag)
		createNewRelease(t, session, "/user2/repo1", "v1.0.0", "v1 Release", false, false)
		createNewRelease(t, session, "/user2/repo1", "v0.1.0", "v0.1 Pre-release", true, false)
		createNewRelease(t, session, "/user2/repo1", "v2.0.0", "v2 Release-Draft", false, true)
		token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository)
		createNewTagUsingAPI(t, token, "user2", "repo1", "v3.0.0", "master", "Tag message")

		// Open Activity page and check stats
		req := NewRequest(t, "GET", "/user2/repo1/activity")
		resp = session.MakeRequest(t, req, http.StatusOK)
		htmlDoc := NewHTMLParser(t, resp.Body)

		// Should be 3 published releases
		list := htmlDoc.doc.Find("#published-releases").Next().Find("p.desc")
		assert.Len(t, list.Nodes, 3)
		var labels []string
		var titles []string
		list.Each(func(i int, s *goquery.Selection) {
			labels = append(labels, s.Find(".label").Text())
			titles = append(titles, s.Find(".title").Text())
		})
		sort.Strings(labels)
		sort.Strings(titles)
		assert.Equal(t, []string{"Pre-release", "Release", "Tag"}, labels)
		assert.Equal(t, []string{"", "v0.1 Pre-release", "v1 Release"}, titles)

		// Should be 1 merged pull request
		list = htmlDoc.doc.Find("#merged-pull-requests").Next().Find("p.desc")
		assert.Len(t, list.Nodes, 1)
		assert.Equal(t, "Merged", list.Find(".label").Text())

		// Should be 2 proposed pull requests
		list = htmlDoc.doc.Find("#proposed-pull-requests").Next().Find("p.desc")
		assert.Len(t, list.Nodes, 2)
		assert.Equal(t, "Proposed", list.Find(".label").First().Text())

		// Should be 0 closed issues
		list = htmlDoc.doc.Find("#closed-issues").Next().Find("p.desc")
		assert.Empty(t, list.Nodes)

		// Should be 3 new issues
		list = htmlDoc.doc.Find("#new-issues").Next().Find("p.desc")
		assert.Len(t, list.Nodes, 3)
		assert.Equal(t, "Opened", list.Find(".label").First().Text())
	})
}

func TestRepoActivityAllUnitsDisabled(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: "user1"})
	session := loginUser(t, user.Name)

	unit_model.LoadUnitConfig()

	// Create a repo, with no unit enabled.
	repo, err := repo_service.CreateRepository(db.DefaultContext, user, user, repo_service.CreateRepoOptions{
		Name:     "empty-repo",
		AutoInit: false,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, repo)

	enabledUnits := make([]repo_model.RepoUnit, 0)
	disabledUnits := []unit_model.Type{unit_model.TypeCode, unit_model.TypeIssues, unit_model.TypePullRequests, unit_model.TypeReleases}
	err = repo_service.UpdateRepositoryUnits(db.DefaultContext, repo, enabledUnits, disabledUnits)
	require.NoError(t, err)

	req := NewRequest(t, "GET", fmt.Sprintf("%s/activity", repo.Link()))
	session.MakeRequest(t, req, http.StatusNotFound)
	req = NewRequest(t, "GET", fmt.Sprintf("%s/activity/contributors", repo.Link()))
	session.MakeRequest(t, req, http.StatusNotFound)
	req = NewRequest(t, "GET", fmt.Sprintf("%s/activity/code-frequency", repo.Link()))
	session.MakeRequest(t, req, http.StatusNotFound)
	req = NewRequest(t, "GET", fmt.Sprintf("%s/activity/recent-commits", repo.Link()))
	session.MakeRequest(t, req, http.StatusNotFound)
}

func TestRepoActivityOnlyCodeUnitWithEmptyRepo(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: "user1"})
	session := loginUser(t, user.Name)

	unit_model.LoadUnitConfig()

	// Create a empty repo, with only code unit enabled.
	repo, err := repo_service.CreateRepository(db.DefaultContext, user, user, repo_service.CreateRepoOptions{
		Name:     "empty-repo",
		AutoInit: false,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, repo)

	enabledUnits := make([]repo_model.RepoUnit, 1)
	enabledUnits[0] = repo_model.RepoUnit{RepoID: repo.ID, Type: unit_model.TypeCode}
	disabledUnits := []unit_model.Type{unit_model.TypeIssues, unit_model.TypePullRequests, unit_model.TypeReleases}
	err = repo_service.UpdateRepositoryUnits(db.DefaultContext, repo, enabledUnits, disabledUnits)
	require.NoError(t, err)

	req := NewRequest(t, "GET", fmt.Sprintf("%s/activity", repo.Link()))
	session.MakeRequest(t, req, http.StatusOK)

	// Git repo empty so no activity for contributors etc
	req = NewRequest(t, "GET", fmt.Sprintf("%s/activity/contributors", repo.Link()))
	session.MakeRequest(t, req, http.StatusNotFound)
	req = NewRequest(t, "GET", fmt.Sprintf("%s/activity/code-frequency", repo.Link()))
	session.MakeRequest(t, req, http.StatusNotFound)
	req = NewRequest(t, "GET", fmt.Sprintf("%s/activity/recent-commits", repo.Link()))
	session.MakeRequest(t, req, http.StatusNotFound)
}

func TestRepoActivityOnlyCodeUnitWithNonEmptyRepo(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: "user1"})
	session := loginUser(t, user.Name)

	unit_model.LoadUnitConfig()

	// Create a repo, with only code unit enabled.
	repo, _, f := tests.CreateDeclarativeRepo(t, user, "", []unit_model.Type{unit_model.TypeCode}, nil, nil)
	defer f()

	req := NewRequest(t, "GET", fmt.Sprintf("%s/activity", repo.Link()))
	session.MakeRequest(t, req, http.StatusOK)

	// Git repo not empty so activity for contributors etc
	req = NewRequest(t, "GET", fmt.Sprintf("%s/activity/contributors", repo.Link()))
	session.MakeRequest(t, req, http.StatusOK)
	req = NewRequest(t, "GET", fmt.Sprintf("%s/activity/code-frequency", repo.Link()))
	session.MakeRequest(t, req, http.StatusOK)
	req = NewRequest(t, "GET", fmt.Sprintf("%s/activity/recent-commits", repo.Link()))
	session.MakeRequest(t, req, http.StatusOK)
}

func TestRepoActivityOnlyIssuesUnit(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: "user1"})
	session := loginUser(t, user.Name)

	unit_model.LoadUnitConfig()

	// Create a empty repo, with only code unit enabled.
	repo, err := repo_service.CreateRepository(db.DefaultContext, user, user, repo_service.CreateRepoOptions{
		Name:     "empty-repo",
		AutoInit: false,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, repo)

	enabledUnits := make([]repo_model.RepoUnit, 1)
	enabledUnits[0] = repo_model.RepoUnit{RepoID: repo.ID, Type: unit_model.TypeIssues}
	disabledUnits := []unit_model.Type{unit_model.TypeCode, unit_model.TypePullRequests, unit_model.TypeReleases}
	err = repo_service.UpdateRepositoryUnits(db.DefaultContext, repo, enabledUnits, disabledUnits)
	require.NoError(t, err)

	req := NewRequest(t, "GET", fmt.Sprintf("%s/activity", repo.Link()))
	session.MakeRequest(t, req, http.StatusOK)

	// Git repo empty so no activity for contributors etc
	req = NewRequest(t, "GET", fmt.Sprintf("%s/activity/contributors", repo.Link()))
	session.MakeRequest(t, req, http.StatusNotFound)
	req = NewRequest(t, "GET", fmt.Sprintf("%s/activity/code-frequency", repo.Link()))
	session.MakeRequest(t, req, http.StatusNotFound)
	req = NewRequest(t, "GET", fmt.Sprintf("%s/activity/recent-commits", repo.Link()))
	session.MakeRequest(t, req, http.StatusNotFound)
}
