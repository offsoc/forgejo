// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"code.gitea.io/gitea/models/db"
	repo_model "code.gitea.io/gitea/models/repo"
	unit_model "code.gitea.io/gitea/models/unit"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/test"
	repo_service "code.gitea.io/gitea/services/repository"
	"code.gitea.io/gitea/tests"

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
		assert.EqualValues(t, "pulls", elem[3])
		testPullMerge(t, session, elem[1], elem[2], elem[4], repo_model.MergeStyleMerge, false)

		testEditFileToNewBranch(t, session, "user1", "repo1", "master", "feat/better_readme", "README.md", "Hello, World (Edited Again)\n")
		testPullCreate(t, session, "user1", "repo1", false, "master", "feat/better_readme", "This is a pull title")

		testEditFileToNewBranch(t, session, "user1", "repo1", "master", "feat/much_better_readme", "README.md", "Hello, World (Edited More)\n")
		testPullCreate(t, session, "user1", "repo1", false, "master", "feat/much_better_readme", "This is a pull title")

		// Create issues (3 new issues)
		testNewIssue(t, session, "user2", "repo1", "Issue 1", "Description 1")
		testNewIssue(t, session, "user2", "repo1", "Issue 2", "Description 2")
		testNewIssue(t, session, "user2", "repo1", "Issue 3", "Description 3")

		// Create releases (1 new release)
		createNewRelease(t, session, "/user2/repo1", "v1.0.0", "v1.0.0", false, false)

		// Open Activity page and check stats
		req := NewRequest(t, "GET", "/user2/repo1/activity")
		resp = session.MakeRequest(t, req, http.StatusOK)
		htmlDoc := NewHTMLParser(t, resp.Body)

		// Should be 1 published release
		list := htmlDoc.doc.Find("#published-releases").Next().Find("p.desc")
		assert.Len(t, list.Nodes, 1)

		// Should be 1 merged pull request
		list = htmlDoc.doc.Find("#merged-pull-requests").Next().Find("p.desc")
		assert.Len(t, list.Nodes, 1)

		// Should be 2 proposed pull requests
		list = htmlDoc.doc.Find("#proposed-pull-requests").Next().Find("p.desc")
		assert.Len(t, list.Nodes, 2)

		// Should be 3 new issues
		list = htmlDoc.doc.Find("#new-issues").Next().Find("p.desc")
		assert.Len(t, list.Nodes, 3)
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
	repo, _, f := CreateDeclarativeRepo(t, user, "", []unit_model.Type{unit_model.TypeCode}, nil, nil)
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
